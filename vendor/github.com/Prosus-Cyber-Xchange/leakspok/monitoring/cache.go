package monitoring

import (
	"context"
)

// TraceCacheGet starts a DataDog APM span for a cache Get operation and returns
// a context with the span attached and a finish function. The caller must
// call the finish function when the operation completes.
func TraceCacheGet(ctx context.Context, entity string) (context.Context, func()) {
	newCtx, span := GlobalTracer().StartSpanFromContext(ctx, "anonymize_cache_get")
	span.SetTag("cache.entity", entity)

	return newCtx, func() {
		span.Finish()
	}
}

// TraceCacheSave starts a DataDog APM span for a cache Save operation and returns
// a context with the span attached and a finish function. The caller must
// call the finish function when the operation completes.
func TraceCacheSave(ctx context.Context, entity string) (context.Context, func()) {
	newCtx, span := GlobalTracer().StartSpanFromContext(ctx, "anonymize_cache_save")
	span.SetTag("cache.entity", entity)

	return newCtx, func() {
		span.Finish()
	}
}
