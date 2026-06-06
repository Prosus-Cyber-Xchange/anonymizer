package anonymizer

import (
	"context"

	ctxpkg "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/context"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
)

// WithRules injects privacy rules into the context for use by middleware/handlers
// Delegates to pkg/context for the actual implementation to avoid circular imports
func WithRules(ctx context.Context, rules []analyzer.Rule) context.Context {
	return ctxpkg.WithRules(ctx, rules)
}

// RulesFromContext extracts privacy rules from the context if available
// Delegates to pkg/context for the actual implementation to avoid circular imports
func RulesFromContext(ctx context.Context) ([]analyzer.Rule, bool) {
	return ctxpkg.RulesFromContext(ctx)
}
