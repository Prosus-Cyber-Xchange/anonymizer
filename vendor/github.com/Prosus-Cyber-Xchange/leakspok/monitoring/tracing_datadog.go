package monitoring

import (
	"context"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
)

// NewDatadogTracer returns a Tracer backed by the DataDog APM SDK.
// Call SetGlobalTracer(NewDatadogTracer()) to enable DataDog tracing.
func NewDatadogTracer() Tracer {
	return &dataDogTracer{}
}

type dataDogTracer struct{}

func (d *dataDogTracer) StartSpanFromContext(ctx context.Context, op string) (context.Context, Span) {
	span, newCtx := tracer.StartSpanFromContext(ctx, op)
	return newCtx, &ddSpan{span: span}
}

type ddSpan struct {
	span *tracer.Span
}

func (s *ddSpan) SetTag(key, value string) {
	s.span.SetTag(key, value)
}

func (s *ddSpan) Finish() {
	s.span.Finish()
}
