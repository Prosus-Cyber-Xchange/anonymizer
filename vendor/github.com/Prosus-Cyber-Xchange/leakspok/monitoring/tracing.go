package monitoring

import (
	"context"
	"sync"
)

// Span represents a single unit of work in a distributed trace.
type Span interface {
	SetTag(key, value string)
	Finish()
}

// Tracer creates spans for distributed tracing.
type Tracer interface {
	StartSpanFromContext(ctx context.Context, operationName string) (context.Context, Span)
}

//nolint:gochecknoglobals // global tracer is intentionally package-level, protected by mutex
var (
	defaultTracerMu sync.RWMutex
	defaultTracer   Tracer = &noopTracer{}
)

// GlobalTracer returns the current package-level tracer.
// It is safe to call from multiple goroutines concurrently.
func GlobalTracer() Tracer {
	defaultTracerMu.RLock()
	defer defaultTracerMu.RUnlock()
	return defaultTracer
}

// SetGlobalTracer replaces the package-level tracer used by
// TracePattern, TraceCacheGet, and TraceCacheSave.
// It is safe to call from multiple goroutines concurrently.
func SetGlobalTracer(t Tracer) {
	defaultTracerMu.Lock()
	defer defaultTracerMu.Unlock()
	defaultTracer = t
}

type noopSpan struct{}

func (noopSpan) SetTag(_ string, _ string) {}
func (noopSpan) Finish()                   {}

type noopTracer struct{}

func (noopTracer) StartSpanFromContext(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}
