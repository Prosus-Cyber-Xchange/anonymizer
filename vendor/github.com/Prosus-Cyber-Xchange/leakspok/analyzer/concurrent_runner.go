package analyzer

import (
	"context"
	"log/slog"

	analyzercache "github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache"
)

// resultItem holds the outcome of a single rule evaluation by a pool worker.
type resultItem struct {
	matched bool
	rule    Rule
}

// ConcurrentRulesRunner is a concurrent implementation of RuleRunner that dispatches
// rule evaluation to a shared goroutine pool.
//
// Multiple goroutines may call Process concurrently. Stop releases the pool;
// subsequent Process calls return immediately with no match.
type ConcurrentRulesRunner struct {
	logger  *slog.Logger
	cache   analyzercache.CacheStore
	options RunnerOptions
	pool    WorkerPool
}

// NewConcurrentRulesRunner creates a ConcurrentRulesRunner backed by the given pool.
// The caller is responsible for the pool's lifecycle; Stop delegates to pool.ReleaseContext.
func NewConcurrentRulesRunner(logger *slog.Logger, options RunnerOptions, cache analyzercache.CacheStore, pool WorkerPool) *ConcurrentRulesRunner {
	return &ConcurrentRulesRunner{
		logger:  logger,
		cache:   cache,
		options: options,
		pool:    pool,
	}
}

// processRule evaluates a single rule against data and sends exactly one result to resultCh.
// resultCh must have sufficient buffer capacity so this send never blocks.
//
// ctx should be a per-call context derived from Process; when another worker finds a
// match first, ctx is cancelled so this worker can bail out before expensive I/O or
// matcher evaluation.
func (r *ConcurrentRulesRunner) processRule(ctx context.Context, rule Rule, data []byte, resultCh chan resultItem) {
	// Early exit if another worker already found a match.
	select {
	case <-ctx.Done():
		resultCh <- resultItem{}
		return
	default:
	}

	// Cache lookup — mirrors SerialRulesRunner.Process cache handling exactly.
	cachedMatch, cacheErr := r.cache.GetMatch(ctx, rule.Matcher.Entity(), data)

	switch {
	case analyzercache.IsCacheNotFoundError(cacheErr):
		r.logger.DebugContext(ctx, "Data and rule not found in cache",
			slog.String("rule_entity", string(rule.Matcher.Entity())),
			slog.String("data", string(data)),
		)
	case cacheErr != nil:
		r.logger.ErrorContext(ctx, "Failed to get matching rule from cache", "error", cacheErr)
	default: // cacheErr == nil: value was cached
		resultCh <- resultItem{matched: cachedMatch, rule: rule}
		return
	}

	// Exception check.
	if isException(ctx, data, rule.Exceptions) {
		resultCh <- resultItem{matched: false, rule: rule}
		return
	}

	// Bail out before the expensive matcher call if a match was already found.
	select {
	case <-ctx.Done():
		resultCh <- resultItem{}
		return
	default:
	}

	// Matcher evaluation.
	matched := rule.Matcher.Match(ctx, data)
	if !matched {
		if err := r.cache.SaveMatch(ctx, rule.Matcher.Entity(), data, matched); err != nil {
			r.logger.ErrorContext(ctx, "Failed to save matching rule in cache", "error", err)
		}
	}

	resultCh <- resultItem{matched: matched, rule: rule}
}

// Process concurrently evaluates rules against data and returns the first matching rule.
//
// Rules with Disable == true are skipped. Process is safe for concurrent use.
// Returns (Rule{}, false) when no rules match, all are disabled, the list is empty,
// the pool has been stopped, or the context is cancelled.
func (r *ConcurrentRulesRunner) Process(ctx context.Context, rules []Rule, data []byte) (Rule, bool) {
	if len(rules) == 0 {
		return Rule{}, false
	}

	// Count enabled rules upfront to size the result buffer.
	var enabledRulesAmount int
	for _, rule := range rules {
		if !rule.Disable {
			enabledRulesAmount++
		}
	}

	if enabledRulesAmount == 0 {
		return Rule{}, false
	}

	// Child context cancelled on first match so remaining workers can bail out
	// before expensive I/O (cache) or matcher evaluation.
	matchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Buffer sized to enabledRulesAmount so workers never block trying to send
	// after the consumer has already found a match or cancelled.
	resultCh := make(chan resultItem, enabledRulesAmount)

	submitted := 0
	for _, rule := range rules {
		if rule.Disable {
			continue
		}

		rule := rule // capture per-iteration value
		err := r.pool.Submit(func() {
			select {
			case <-matchCtx.Done():
				resultCh <- resultItem{}
				return
			default:
			}
			r.processRule(matchCtx, rule, data, resultCh)
		})
		if err != nil {
			// Pool is closed — return without match.
			// Already-submitted closures will write to the buffered resultCh and exit cleanly.
			return Rule{}, false
		}

		submitted++
	}

	// Collect results for all submitted tasks.
	for range submitted {
		select {
		case result := <-resultCh:
			if result.matched {
				cancel() // signal remaining workers to stop
				return result.rule, true
			}
		case <-matchCtx.Done():
			return Rule{}, false
		}
	}

	return Rule{}, false
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
