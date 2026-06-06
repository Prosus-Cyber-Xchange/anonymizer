package server

import (
	"log/slog"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/uber-go/tally/v4"
)

// Option configures the AnonymizerServer during construction.
type Option func(*AnonymizerServer)

// WithPlugin registers a plugin. The builder detects MiddlewareRegistrar
// via type assertion and wires it accordingly.
func WithPlugin(p any) Option {
	return func(a *AnonymizerServer) {
		a.plugins = append(a.plugins, p)
	}
}

// WithLogger sets a custom logger. If not called, slog.Default() is used.
func WithLogger(l *slog.Logger) Option {
	return func(a *AnonymizerServer) {
		a.logger = l
	}
}

// WithByteAnalyzer sets a custom ByteAnalyzer. If not called, one is
// created internally from environment configuration.
func WithByteAnalyzer(ba analyzer.ByteAnalyzer) Option {
	return func(a *AnonymizerServer) {
		a.byteAnalyzer = &ba
	}
}

// WithMetricsScope sets a tally.Scope for metrics collection.
// If not called, metrics are disabled (NoopScope).
func WithMetricsScope(scope tally.Scope) Option {
	return func(a *AnonymizerServer) {
		a.metricsScope = scope
	}
}
