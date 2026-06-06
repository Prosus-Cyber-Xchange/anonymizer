package monitoring

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "anonymizer-service"

// Span wraps an OTel span to provide a uniform interface.
type Span struct {
	span trace.Span
}

// Finish ends the span.
func (s *Span) Finish() {
	if s == nil || s.span == nil {
		return
	}
	s.span.End()
}

// StartSpan creates and starts a new span in the current trace.
func StartSpan(ctx context.Context, operationName string) (*Span, context.Context) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, operationName)
	return &Span{span: span}, ctx
}

// SetError records an error on the span.
func SetError(span *Span, err error) {
	if span == nil || span.span == nil || err == nil {
		return
	}
	span.span.RecordError(err)
	span.span.SetStatus(codes.Error, err.Error())
}

// SetTag sets a key-value attribute on the span.
func SetTag(span *Span, key string, value interface{}) {
	if span == nil || span.span == nil {
		return
	}
	switch v := value.(type) {
	case string:
		span.span.SetAttributes(attribute.String(key, v))
	case int:
		span.span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.span.SetAttributes(attribute.Int64(key, v))
	case bool:
		span.span.SetAttributes(attribute.Bool(key, v))
	case float64:
		span.span.SetAttributes(attribute.Float64(key, v))
	}
}

// SetTags sets multiple attributes on a span.
func SetTags(span *Span, tags map[string]interface{}) {
	for key, value := range tags {
		SetTag(span, key, value)
	}
}

// FinishWithError finishes a span and tags it with error information.
func FinishWithError(span *Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		SetError(span, err)
	}
	span.Finish()
}

// FinishSpan finishes a span
func FinishSpan(span *Span) {
	if span == nil {
		return
	}
	span.Finish()
}
