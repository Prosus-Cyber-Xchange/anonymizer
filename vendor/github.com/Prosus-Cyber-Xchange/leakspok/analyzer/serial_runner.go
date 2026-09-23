package analyzer

import (
	"context"
	"log/slog"

	analyzercache "github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache"
)

// SerialRulesRunner evaluates rules one at a time in order and returns the first match.
// It supports optional caching of match results to avoid repeated expensive
// matcher evaluations for the same entity+data combination.
//
// SerialRulesRunner is safe for concurrent use.
type SerialRulesRunner struct {
	logger    *slog.Logger
	cache     analyzercache.CacheStore
	coalescer singleflightDoer
}

// NewSerialRulesRuner creates a new instance of SerialRulesRunner with the provided
// logger, runner options, and cache store.
func NewSerialRulesRuner(logger *slog.Logger, options RunnerOptions, cache analyzercache.CacheStore) SerialRulesRunner {
	var coalescer singleflightDoer = disabledSingleflightCoalescer{}
	if options.Cache.Enabled && options.Cache.SingleflightEnabled {
		coalescer = newSingleflightCoalescer()
	}

	return SerialRulesRunner{
		logger:    logger,
		cache:     cache,
		coalescer: coalescer,
	}
}

// Process evaluates each enabled rule sequentially against data and returns the first
// matching rule. Rules with Disable set to true are skipped. When caching is enabled,
// negative results are cached to avoid re-running the expensive matcher.
// Returns (Rule{}, false) when no rules match.
func (s SerialRulesRunner) Process(ctx context.Context, rules []Rule, data []byte) (Rule, bool) {
	for _, rule := range rules {
		if rule.Disable {
			continue
		}

		cachedMatch, cacheErr := s.cache.GetMatch(ctx, rule.Matcher.Entity(), data)
		switch {
		case analyzercache.IsCacheNotFoundError(cacheErr):
			s.logger.DebugContext(ctx, "Data and rule not found in cache",
				slog.String("rule_entity", string(rule.Matcher.Entity())),
				slog.String("data", string(data)),
			)
		case cacheErr != nil:
			s.logger.ErrorContext(ctx, "Failed to get matching rule from cache", "error", cacheErr)
		default: // cacheErr is nil, value was cached
			if !cachedMatch {
				continue
			}

			return rule, cachedMatch
		}

		if isException(ctx, data, rule.Exceptions) {
			continue
		}

		// Coalesce identical concurrent misses for the same entity+data key.
		// The exception check stays outside the flight; the matcher and the
		// negative-result save run inside it so a burst computes and saves once.
		key := singleflightKey(string(rule.Matcher.Entity()), data)
		matched, coalesceErr := s.coalescer.do(ctx, key, func() (bool, error) {
			return s.matchAndSave(ctx, rule, data)
		})
		if coalesceErr != nil {
			// A cancelled waiter never computes; a failed negative-result save
			// is reported and the rule is skipped, matching the log-and-continue
			// convention.
			s.logger.ErrorContext(ctx, "Failed to save matching rule in cache", "error", coalesceErr)
			continue
		}

		if !matched {
			continue
		}

		// Stop checking remaining rule once first match found
		return rule, true
	}

	return Rule{}, false
}

// matchAndSave runs the matcher and persists a negative result when it does
// not match, returning the computation result for the coalescer.
func (s SerialRulesRunner) matchAndSave(ctx context.Context, rule Rule, data []byte) (bool, error) {
	matched := rule.Matcher.Match(ctx, data)
	if !matched {
		if err := s.cache.SaveMatch(ctx, rule.Matcher.Entity(), data, matched); err != nil {
			return false, err
		}
	}

	return matched, nil
}

// Stop is a no-op for SerialRulesRunner (it has no resources to release).
// It exists to satisfy the RuleRunner interface.
func (s SerialRulesRunner) Stop() {}
