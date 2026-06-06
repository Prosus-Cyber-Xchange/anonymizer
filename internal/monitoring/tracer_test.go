package monitoring

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStartSpan(t *testing.T) {
	ctx := context.Background()

	span, newCtx := StartSpan(ctx, "test-operation")
	// span should not be nil
	assert.NotNil(t, span)
	assert.NotNil(t, newCtx)

	// Context should be different after creating a span
	assert.NotEqual(t, ctx, newCtx)

	// FinishSpan should handle nil spans gracefully
	FinishSpan(span)
}

func TestSetError(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")
	defer span.Finish()

	testErr := errors.New("test error")
	// Should not panic even if span is nil
	SetError(span, testErr)
}

func TestSetError_NilSpan(t *testing.T) {
	// Should not panic when span is nil
	var nilSpan *Span
	SetError(nilSpan, errors.New("test error"))
}

func TestSetError_NilError(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")
	defer span.Finish()

	// Should not panic when error is nil
	SetError(span, nil)
}

func TestSetTag(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")
	defer span.Finish()

	// Should not panic even if span is nil
	SetTag(span, "test-key", "test-value")
}

func TestSetTag_NilSpan(t *testing.T) {
	// Should not panic when span is nil
	var nilSpan *Span
	SetTag(nilSpan, "test-key", "test-value")
}

func TestSetTags(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")
	defer span.Finish()

	tags := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	// Should not panic even if span is nil
	SetTags(span, tags)
}

func TestSetTags_NilSpan(t *testing.T) {
	// Should not panic when span is nil
	var nilSpan *Span
	tags := map[string]interface{}{
		"key1": "value1",
	}
	SetTags(nilSpan, tags)
}

func TestFinishWithError(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")

	testErr := errors.New("test error")
	// Should complete without panic even if span is nil
	FinishWithError(span, testErr)
}

func TestFinishWithError_NoError(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")

	// Should complete without panic even if span is nil
	FinishWithError(span, nil)
}

func TestFinishWithError_NilSpan(t *testing.T) {
	// Should not panic when span is nil
	var nilSpan *Span
	FinishWithError(nilSpan, errors.New("test error"))
}

func TestFinishSpan(t *testing.T) {
	ctx := context.Background()
	span, _ := StartSpan(ctx, "test-operation")

	// Should complete without panic even if span is nil
	FinishSpan(span)
}

func TestFinishSpan_NilSpan(t *testing.T) {
	// Should not panic when span is nil
	var nilSpan *Span
	FinishSpan(nilSpan)
}

func TestSpanContextPropagation(t *testing.T) {
	ctx := context.Background()

	// Create first span
	span1, ctx1 := StartSpan(ctx, "operation1")
	defer span1.Finish()

	// Create second span from first span's context
	span2, ctx2 := StartSpan(ctx1, "operation2")
	defer span2.Finish()

	// Both contexts should be different from original
	assert.NotEqual(t, ctx, ctx1)
	assert.NotEqual(t, ctx1, ctx2)
}
