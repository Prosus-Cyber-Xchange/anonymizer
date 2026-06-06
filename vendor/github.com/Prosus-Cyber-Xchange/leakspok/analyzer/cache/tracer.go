package cache

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/monitoring"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

var _ CacheStore = (*CacheStoreTracer)(nil)

// CacheStoreTracer is a decorator for CacheStore that adds DataDog APM tracing.
// Each GetMatch and SaveMatch call emits a DataDog span with a cache.entity tag.
type CacheStoreTracer struct {
	underlying CacheStore
}

// NewCacheStoreTracer wraps a CacheStore with DataDog APM tracing.
func NewCacheStoreTracer(underlying CacheStore) *CacheStoreTracer {
	return &CacheStoreTracer{underlying: underlying}
}

// GetMatch starts a DataDog trace span and delegates to the underlying cache,
// forwarding the span-bearing context so the cache operation nests under the span.
func (c *CacheStoreTracer) GetMatch(ctx context.Context, entity pattern.Entity, data []byte) (bool, error) {
	newCtx, finish := monitoring.TraceCacheGet(ctx, string(entity))
	defer finish()

	return c.underlying.GetMatch(newCtx, entity, data)
}

// SaveMatch starts a DataDog trace span and delegates to the underlying cache,
// forwarding the span-bearing context so the cache operation nests under the span.
func (c *CacheStoreTracer) SaveMatch(ctx context.Context, entity pattern.Entity, data []byte, matched bool) error {
	newCtx, finish := monitoring.TraceCacheSave(ctx, string(entity))
	defer finish()

	return c.underlying.SaveMatch(newCtx, entity, data, matched)
}
