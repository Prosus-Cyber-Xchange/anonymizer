package analyzer

import (
	"context"
	"log/slog"
	"maps"

	analyzercache "github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// SerialRulesRunner evaluates rules one at a time in order and returns the first match.
// It supports optional caching of match results to avoid repeated expensive
// matcher evaluations for the same token.
//
// SerialRulesRunner is safe for concurrent use.
type SerialRulesRunner struct {
	logger    *slog.Logger
	cache     analyzercache.CacheStore
	coalescer tokenFlightDoer
}

// NewSerialRulesRuner creates a new instance of SerialRulesRunner with the provided
// logger, runner options, and cache store.
func NewSerialRulesRuner(logger *slog.Logger, options RunnerOptions, cache analyzercache.CacheStore) SerialRulesRunner {
	var coalescer tokenFlightDoer = disabledSingleflightCoalescer{}
	if options.Cache.Enabled && options.Cache.SingleflightEnabled {
		coalescer = newSingleflightCoalescer()
	}

	return SerialRulesRunner{
		logger:    logger,
		cache:     cache,
		coalescer: coalescer,
	}
}

// missingEvaluation is one entity of a token's missing set together with the
// first rule claiming it, whose matcher performs the single evaluation shared
// by every rule with the same entity.
type missingEvaluation struct {
	entity pattern.Entity
	rule   Rule
}

// ruleView is the per-rule classification built from one token read: where the
// rule's outcome comes from in the final in-order pass.
type ruleView struct {
	rule     Rule
	entity   pattern.Entity
	cached   bool // entity present in the token's cached map; view.result holds its value
	result   bool
	excepted bool // the rule's own exception matched; never evaluated or persisted
}

// Process evaluates each enabled rule sequentially against data and returns the first
// matching rule. Rules with Disable set to true are skipped. When caching is enabled,
// the token's cached results are read once, only missing entities are evaluated after
// their own exception checks, and negative results are persisted in one batched save.
// Returns (Rule{}, false) when no rules match.
func (s SerialRulesRunner) Process(ctx context.Context, rules []Rule, data []byte) (Rule, bool) {
	cached, cacheErr := s.cache.GetMatches(ctx, data)
	if cacheErr != nil {
		// Degrade to full evaluation: treat every entity as a cache miss.
		s.logger.ErrorContext(ctx, "Failed to get matching rules from cache", "error", cacheErr)
		cached = map[pattern.Entity]bool{}
	}

	views, missing := s.classifyRules(ctx, rules, data, cached)

	evaluated := map[pattern.Entity]bool{}
	if len(missing) > 0 {
		evaluated = s.evaluateMissing(ctx, data, missing)
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
func (s SerialRulesRunner) classifyRules(ctx context.Context, rules []Rule, data []byte, cached map[pattern.Entity]bool) ([]ruleView, []missingEvaluation) {
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

// evaluateMissing evaluates the token's missing entities and persists their
// negative results. When coalescing is enabled, the evaluation and the batched
// save run inside a token-level flight: the leader evaluates its own missing
// set in rule order and persists one negative batch, while waiters share the
// results and the save error and evaluate only their residual misses inline,
// so each caller issues at most one SaveMatches.
func (s SerialRulesRunner) evaluateMissing(ctx context.Context, data []byte, missing []missingEvaluation) map[pattern.Entity]bool {
	results, flightErr := s.coalescer.doToken(ctx, tokenFlightKey(data), func() (map[pattern.Entity]bool, error) {
		// Leader path: evaluate the missing set in rule order and persist
		// one negative batch. The save error is returned together with the
		// results so waiters still receive the shared computation.
		results := s.matchMissing(ctx, data, missing)
		return results, s.saveNegatives(ctx, data, results)
	})

	if results == nil {
		// A cancelled waiter unblocks with its context error while the shared
		// computation continues for the remaining callers.
		s.logger.ErrorContext(ctx, "Cancelled while waiting for token evaluation", "error", flightErr)
		return map[pattern.Entity]bool{}
	}

	if flightErr != nil {
		// A failed negative-result save is shared, logged, and continued past
		// with the computed results.
		s.logger.ErrorContext(ctx, "Failed to save matching rules in cache", "error", flightErr)
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
		return results
	}

	residualResults := s.matchMissing(ctx, data, residual)
	if saveErr := s.saveNegatives(ctx, data, residualResults); saveErr != nil {
		s.logger.ErrorContext(ctx, "Failed to save matching rules in cache", "error", saveErr)
	}

	// The shared results map is read-only for waiters (concurrent callers hold
	// the same instance), so merged outcomes go into a fresh map.
	merged := make(map[pattern.Entity]bool, len(results)+len(residualResults))
	maps.Copy(merged, results)
	maps.Copy(merged, residualResults)
	return merged
}

// matchMissing evaluates each missing entity once, in rule order, using the
// matcher of the first rule claiming the entity.
func (s SerialRulesRunner) matchMissing(ctx context.Context, data []byte, missing []missingEvaluation) map[pattern.Entity]bool {
	results := make(map[pattern.Entity]bool, len(missing))
	for _, m := range missing {
		results[m.entity] = m.rule.Matcher.Match(ctx, data)
	}
	return results
}

// saveNegatives persists the negative results of one evaluation in a single
// batched save. The canonical invariant is enforced here: positive matches are
// never cached, so an all-positive batch saves nothing.
func (s SerialRulesRunner) saveNegatives(ctx context.Context, data []byte, results map[pattern.Entity]bool) error {
	negatives := make(map[pattern.Entity]bool, len(results))
	for entity, matched := range results {
		if !matched {
			negatives[entity] = false
		}
	}
	if len(negatives) == 0 {
		return nil
	}

	return s.cache.SaveMatches(ctx, data, negatives)
}

// Stop is a no-op for SerialRulesRunner (it has no resources to release).
// It exists to satisfy the RuleRunner interface.
func (s SerialRulesRunner) Stop() {}
