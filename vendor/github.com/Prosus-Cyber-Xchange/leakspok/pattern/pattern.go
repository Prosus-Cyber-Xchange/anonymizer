// Package pattern provides interfaces and functions for creating and combining
// pattern matchers that operate on byte slices. It supports logical operations
// like AND, OR, NOT, and threshold-based matching to build complex pattern
// matching rules for security analysis and leak detection.
package pattern

import (
	"context"
	"reflect"
	"runtime"
	"strings"

	"github.com/Prosus-Cyber-Xchange/leakspok/monitoring"
)

//nolint:revive // type name is fine
type PatternPredicate func(context.Context, []byte) bool

// Pattern defines the interface for all pattern matchers.
// Implementations should return true when the input matches the pattern criteria.
type Pattern interface {
	Name() string
	// Match returns true if the input byte slice matches the pattern.
	Match(ctx context.Context, input []byte) bool
}

func getFunctionName(f interface{}) string {
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	parts := strings.Split(fullName, "/")
	return parts[len(parts)-1]
}

// PatternFunc creates a named Pattern from a plain function. The name is derived from the function symbol.
//
//nolint:revive // type name is fine
func PatternFunc(f PatternPredicate) Pattern {
	return NewBasePattern(getFunctionName(f), f)
}

// BasePattern is a named pattern that uses a predicate function to determine
// whether the input matches. When pattern monitoring is enabled in the context,
// BasePattern automatically emits tracing spans and debug logs around each
// Match call.
type BasePattern struct {
	name string
	f    PatternPredicate
}

// NewBasePattern creates a BasePattern with the given name and predicate function.
// The name is used in tracing spans and log messages; the predicate determines
// whether the input matches the pattern.
func NewBasePattern(name string, f PatternPredicate) BasePattern {
	return BasePattern{
		name: name,
		f:    f,
	}
}

// Name returns the pattern name, used for tracing and logging identification.
func (p BasePattern) Name() string {
	return p.name
}

// Match implements the Pattern interface for BasePattern.
// Pattern tracing (spans and debug logging) is only performed when the context
// has tracing enabled via monitoring.WithPatternMonitoring.
func (p BasePattern) Match(ctx context.Context, input []byte) bool {
	if !monitoring.PatternMonitoringEnabled(ctx) {
		return p.f(ctx, input)
	}

	newCtx, traceFinish := monitoring.TracePattern(ctx, p.name)
	defer traceFinish()

	monitoring.LogPatternStart(newCtx, p.name)
	output := p.f(newCtx, input)
	monitoring.LogPatternEnd(newCtx, p.name, output)

	return output
}
