package analyzer

import (
	"context"
	"errors"
	"log/slog"
	"maps"

	analyzercache "github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// errPoolClosed aborts a token's evaluation when the worker pool rejects a
// submission: the runner is shutting down, so no results are produced and the
// caller reports no match.
var errPoolClosed = errors.New("analyzer: worker pool closed")

// resultItem holds the outcome of one missing-entity evaluation by a pool
// worker. Workers bailed out by first-match cancellation send the zero item
// (evaluated false), which the collector skips.
type resultItem struct {
	rule      Rule
	entity    pattern.Entity
	matched   bool
	evaluated bool
}

// ConcurrentRulesRunner is a concurrent implementation of RuleRunner that dispatches
// rule evaluation to a shared goroutine pool.
//
// Multiple goroutines may call Process concurrently. Stop releases the pool;
// subsequent Process calls return immediately with no match.
type ConcurrentRulesRunner struct {
	logger    *slog.Logger
	cache     analyzercache.CacheStore
	pool      WorkerPool
	coalescer tokenFlightDoer
}

// NewConcurrentRulesRunner creates a ConcurrentRulesRunner backed by the given pool.
// The caller is responsible for the pool's lifecycle; Stop delegates to pool.ReleaseContext.
func NewConcurrentRulesRunner(logger *slog.Logger, options RunnerOptions, cache analyzercache.CacheStore, pool WorkerPool) *ConcurrentRulesRunner {
	var coalescer tokenFlightDoer = disabledSingleflightCoalescer{}
	if options.Cache.Enabled && options.Cache.SingleflightEnabled {
		coalescer = newSingleflightCoalescer()
	}

	return &ConcurrentRulesRunner{
		logger:    logger,
		cache:     cache,
		pool:      pool,
		coalescer: coalescer,
	}
}

// Process concurrently evaluates rules against data and returns the first matching rule.
// Rules with Disable == true are skipped. The token's cached results are read once
// before any worker is submitted, only missing entities are submitted for evaluation,
// and the negatives of completed workers are persisted in one batched save. Process
// returns as soon as a collected result matches; the remaining workers are cancelled
// by the first-match signal and produce nothing.
// Process is safe for concurrent use.
// Returns (Rule{}, false) when no rules match, all are disabled, the list is empty,
// the pool has been stopped, or the context is cancelled.
func (r *ConcurrentRulesRunner) Process(ctx context.Context, rules []Rule, data []byte) (Rule, bool) {
	if len(rules) == 0 {
		return Rule{}, false
	}

	// One token-scoped read per Process call, before submitting workers.
	cached, cacheErr := r.cache.GetMatches(ctx, data)
	if cacheErr != nil {
		// Degrade to full evaluation: treat every entity as a cache miss.
		r.logger.ErrorContext(ctx, "Failed to get matching rules from cache", "error", cacheErr)
		cached = map[pattern.Entity]bool{}
	}

	views, missing := r.classifyRules(ctx, rules, data, cached)

	evaluated := map[pattern.Entity]bool{}
	if len(missing) > 0 {
		var winner Rule
		var found bool
		evaluated, winner, found = r.evaluateMissing(ctx, data, missing)
		if found {
			// First-match short-circuit: the collector returned as soon as a
			// result matched, and in-flight workers were cancelled.
			return winner, true
		}
	}

	// Final in-order pass: the first rule whose entity result is true wins,
	// preserving first-match semantics.
	for _, view := range views {
		switch {
		case view.cached:
			if view.result {
				return view.rule, true
			}
		case view.excepted:
			// Excepted entities are never evaluated or persisted.
		default:
			if evaluated[view.entity] {
				return view.rule, true
			}
		}
	}

	return Rule{}, false
}

// classifyRules builds the ordered rule view for the token: rules whose entity
// is cached use the stored value, rules whose own exception matches are
// skipped without evaluation, and every other rule contributes its entity to
// the missing set, deduplicated by entity in rule order.
func (r *ConcurrentRulesRunner) classifyRules(ctx context.Context, rules []Rule, data []byte, cached map[pattern.Entity]bool) ([]ruleView, []missingEvaluation) {
	views := make([]ruleView, 0, len(rules))
	missing := make([]missingEvaluation, 0, len(rules))
	queued := make(map[pattern.Entity]struct{}, len(rules))

	for _, rule := range rules {
		if rule.Disable {
			continue
		}

		view := ruleView{rule: rule, entity: rule.Matcher.Entity()}
		if result, ok := cached[view.entity]; ok {
			view.cached = true
			view.result = result
		} else if isException(ctx, data, rule.Exceptions) {
			view.excepted = true
		} else if _, seen := queued[view.entity]; !seen {
			queued[view.entity] = struct{}{}
			missing = append(missing, missingEvaluation{entity: view.entity, rule: rule})
		}
		views = append(views, view)
	}

	return views, missing
}

// evaluateMissing evaluates the token's missing entities on the worker pool
// and persists their negative results. When coalescing is enabled, the whole
// evaluate-and-save phase runs inside a token-level flight: the leader submits
// the workers, collects the results, and persists one negative batch, while
// waiters hold no pool slots, share the results and the save error, and
// evaluate only their residual misses inline sequentially.
func (r *ConcurrentRulesRunner) evaluateMissing(ctx context.Context, data []byte, missing []missingEvaluation) (map[pattern.Entity]bool, Rule, bool) {
	var winner Rule
	var found bool

	results, flightErr := r.coalescer.doToken(ctx, tokenFlightKey(data), func() (map[pattern.Entity]bool, error) {
		// Leader path: submit one pool task per missing entity, collect with
		// the existing first-match cancellation, and persist the completed
		// workers' negatives in one batched save. The save error is returned
		// together with the results so waiters still receive the shared
		// computation.
		results, rule, matched, saveErr := r.collectEvaluations(ctx, data, missing)
		winner, found = rule, matched
		return results, saveErr
	})

	if results == nil {
		// A cancelled waiter unblocks with its context error while the shared
		// computation continues for the remaining callers; a closed pool
		// aborts evaluation for every caller. Either way the caller resumes
		// with no results and no match.
		r.logger.ErrorContext(ctx, "Token evaluation did not complete", "error", flightErr)
		return map[pattern.Entity]bool{}, Rule{}, false
	}

	if flightErr != nil {
		// A failed negative-result save is shared, logged, and continued past
		// with the computed results.
		r.logger.ErrorContext(ctx, "Failed to save matching rules in cache", "error", flightErr)
	}

	if found {
		return results, winner, true
	}

	if ctx.Err() != nil {
		// The caller's context cancelled the collection: in-flight workers
		// were abandoned, so there is no residual evaluation to complete and
		// the caller reports no match.
		return results, Rule{}, false
	}

	// A waiter's missing set may exceed the leader's: evaluate residual
	// misses inline and persist their negatives with at most one save.
	residual := make([]missingEvaluation, 0, len(missing))
	for _, m := range missing {
		if _, ok := results[m.entity]; !ok {
			residual = append(residual, m)
		}
	}
	if len(residual) == 0 {
		return results, Rule{}, false
	}

	residualResults := r.matchMissing(ctx, data, residual)
	if saveErr := r.saveNegatives(ctx, data, residualResults); saveErr != nil {
		r.logger.ErrorContext(ctx, "Failed to save matching rules in cache", "error", saveErr)
	}

	// The shared results map is read-only for waiters (concurrent callers hold
	// the same instance), so merged outcomes go into a fresh map.
	merged := make(map[pattern.Entity]bool, len(results)+len(residualResults))
	maps.Copy(merged, results)
	maps.Copy(merged, residualResults)
	return merged, Rule{}, false
}

// collectEvaluations submits one pool task per missing entity and collects
// their outcomes, preserving the existing first-match cancellation: the
// collection stops at the first matching result, remaining workers are
// cancelled and produce nothing, and the negatives of completed workers are
// persisted in one batched save. Returns the evaluated outcomes, the matching
// rule when a match was collected, and the save error if any.
func (r *ConcurrentRulesRunner) collectEvaluations(ctx context.Context, data []byte, missing []missingEvaluation) (map[pattern.Entity]bool, Rule, bool, error) {
	// Child context cancelled on first match so remaining workers can bail out
	// before expensive matcher evaluation.
	matchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Buffer sized to the missing set so workers never block trying to send
	// after the collector has already found a match or returned.
	resultCh := make(chan resultItem, len(missing))

	submitted := 0
	for _, m := range missing {
		m := m // capture per-iteration value
		err := r.pool.Submit(func() {
			select {
			case <-matchCtx.Done():
				resultCh <- resultItem{}
				return
			default:
			}
			resultCh <- resultItem{rule: m.rule, entity: m.entity, matched: m.rule.Matcher.Match(matchCtx, data), evaluated: true}
		})
		if err != nil {
			// Pool is closed — abort the token's evaluation without results.
			// Already-submitted closures will write to the buffered resultCh and exit cleanly.
			return nil, Rule{}, false, errPoolClosed
		}

		submitted++
	}

	results := make(map[pattern.Entity]bool, len(missing))
	for range submitted {
		select {
		case result := <-resultCh:
			if !result.evaluated {
				continue
			}
			results[result.entity] = result.matched
			if result.matched {
				cancel() // signal remaining workers to stop
				r.drainCompleted(resultCh, results)
				return results, result.rule, true, r.saveNegatives(ctx, data, results)
			}
		case <-matchCtx.Done():
			// Caller cancelled: return the collected outcomes without
			// persisting anything, matching the previous cancellation profile.
			return results, Rule{}, false, nil
		}
	}

	// Every worker completed without a match: persist all negatives.
	return results, Rule{}, false, r.saveNegatives(ctx, data, results)
}

// drainCompleted collects the outcomes of workers that finished while the
// collector was returning the match: their sends are already buffered, so a
// non-blocking drain captures every completed worker's outcome without
// awaiting the in-flight ones.
func (r *ConcurrentRulesRunner) drainCompleted(resultCh chan resultItem, results map[pattern.Entity]bool) {
	for {
		select {
		case result := <-resultCh:
			if result.evaluated {
				results[result.entity] = result.matched
			}
		default:
			return
		}
	}
}

// matchMissing evaluates each missing entity once, in rule order, using the
// matcher of the first rule claiming the entity.
func (r *ConcurrentRulesRunner) matchMissing(ctx context.Context, data []byte, missing []missingEvaluation) map[pattern.Entity]bool {
	results := make(map[pattern.Entity]bool, len(missing))
	for _, m := range missing {
		results[m.entity] = m.rule.Matcher.Match(ctx, data)
	}
	return results
}

// saveNegatives persists the negative results of one evaluation in a single
// batched save. The canonical invariant is enforced here: positive matches are
// never cached, so an all-positive batch saves nothing.
func (r *ConcurrentRulesRunner) saveNegatives(ctx context.Context, data []byte, results map[pattern.Entity]bool) error {
	negatives := make(map[pattern.Entity]bool, len(results))
	for entity, matched := range results {
		if !matched {
			negatives[entity] = false
		}
	}
	if len(negatives) == 0 {
		return nil
	}

	return r.cache.SaveMatches(ctx, data, negatives)
}

// Stop releases the pool. Subsequent calls to Process return immediately.
// Stop is idempotent and safe to call concurrently with Process.
//
// Stop returns without waiting for in-flight tasks to complete. The pool's
// own lifecycle ensures workers exit cleanly after their current task finishes.
func (r *ConcurrentRulesRunner) Stop() {
	// Use a pre-cancelled context so ReleaseContext marks the pool as closed
	// and returns immediately, without blocking on in-flight workers.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = r.pool.ReleaseContext(ctx)
}
