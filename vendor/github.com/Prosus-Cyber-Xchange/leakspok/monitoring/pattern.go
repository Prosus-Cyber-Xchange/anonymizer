package monitoring

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type loggerCtxKey struct{}
type workIDCtxKey struct{}
type patternMonitoringEnabledCtxKey struct{}

// WithLogger returns a new context with the provided logger stored in it
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, logger)
}

// LoggerFromContext retrieves the logger from the context.
// If no logger is found, it returns the default logger.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// WithWorkID creates a new UUID and returns a new context with the work ID stored in it
func WithWorkID(ctx context.Context) context.Context {
	workID := uuid.New().String()
	return context.WithValue(ctx, workIDCtxKey{}, workID)
}

// WorkIDFromContext retrieves the work ID from the context.
// If no work ID is found, it returns an empty string.
func WorkIDFromContext(ctx context.Context) string {
	if workID, ok := ctx.Value(workIDCtxKey{}).(string); ok {
		return workID
	}
	return ""
}

// WithPatternMonitoring returns a new context that enables or disables pattern tracing.
func WithPatternMonitoring(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, patternMonitoringEnabledCtxKey{}, enabled)
}

// PatternMonitoringEnabled reports whether pattern tracing is enabled in the context.
// It returns false when no value has been set.
func PatternMonitoringEnabled(ctx context.Context) bool {
	enabled, ok := ctx.Value(patternMonitoringEnabledCtxKey{}).(bool)
	return ok && enabled
}

// LogPatternStart emits a debug log entry indicating that pattern matching has
// begun for the given pattern name. The logger and work ID are extracted from
// the provided context.
func LogPatternStart(ctx context.Context, patternName string) {
	logger := LoggerFromContext(ctx)
	id := WorkIDFromContext(ctx)

	logger.DebugContext(
		ctx,
		"Pattern match started",
		slog.String("pattern", patternName),
		slog.String("workID", id),
	)
}

// LogPatternEnd emits a debug log entry indicating that pattern matching has
// completed, including whether the pattern matched. The logger and work ID are
// extracted from the provided context.
func LogPatternEnd(ctx context.Context, patternName string, output bool) {
	logger := LoggerFromContext(ctx)
	id := WorkIDFromContext(ctx)

	logger.DebugContext(
		ctx,
		"Pattern match ended",
		slog.String("pattern", patternName),
		slog.Bool("matched", output),
		slog.String("workID", id),
	)
}

// TracePattern starts a DataDog APM span for pattern matching and returns a
// context with the span attached and a finish function. The caller must call
// the finish function when the pattern evaluation completes.
func TracePattern(ctx context.Context, patternName string) (context.Context, func()) {
	newCtx, span := GlobalTracer().StartSpanFromContext(ctx, patternName)

	return newCtx, func() {
		span.Finish()
	}
}
