package context

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
)

type rulesContextKey struct{}

// WithRules injects privacy rules into the context for use by middleware/handlers
func WithRules(ctx context.Context, rules []analyzer.Rule) context.Context {
	return context.WithValue(ctx, rulesContextKey{}, rules)
}

// RulesFromContext extracts privacy rules from the context if available
func RulesFromContext(ctx context.Context) ([]analyzer.Rule, bool) {
	rules, ok := ctx.Value(rulesContextKey{}).([]analyzer.Rule)
	return rules, ok
}
