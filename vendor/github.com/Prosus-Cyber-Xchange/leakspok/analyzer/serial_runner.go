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
	logger *slog.Logger
	cache  analyzercache.CacheStore
}

// NewSerialRulesRuner creates a new instance of SerialRulesRunner with the provided logger and cache store.
func NewSerialRulesRuner(logger *slog.Logger, cache analyzercache.CacheStore) SerialRulesRunner {
	return SerialRulesRunner{
		logger: logger,
		cache:  cache,
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

		matched := rule.Matcher.Match(ctx, data)
		if !matched {
			err := s.cache.SaveMatch(ctx, rule.Matcher.Entity(), data, matched)
			if err != nil {
				s.logger.ErrorContext(ctx, "Failed to save matching rule in cache", "error", err)
			}

			continue
		}

		// Stop checking remaining rule once first match found
		return rule, true
	}

	return Rule{}, false
}

// Stop is a no-op for SerialRulesRunner (it has no resources to release).
// It exists to satisfy the RuleRunner interface.
func (s SerialRulesRunner) Stop() {}
