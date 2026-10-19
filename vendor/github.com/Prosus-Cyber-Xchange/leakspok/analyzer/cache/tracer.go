package cache

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/monitoring"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

var _ CacheStore = (*CacheStoreTracer)(nil)

// CacheStoreTracer is a decorator for CacheStore that adds DataDog APM tracing.
// Each GetMatches and SaveMatches call emits a DataDog span with a cache.token tag.
type CacheStoreTracer struct {
	underlying CacheStore
}

// NewCacheStoreTracer wraps a CacheStore with DataDog APM tracing.
func NewCacheStoreTracer(underlying CacheStore) *CacheStoreTracer {
	return &CacheStoreTracer{underlying: underlying}
}

// GetMatches starts a DataDog trace span and delegates to the underlying cache,
// forwarding the span-bearing context so the token-scoped cache operation
// nests under the span.
func (c *CacheStoreTracer) GetMatches(ctx context.Context, data []byte) (map[pattern.Entity]bool, error) {
	newCtx, finish := monitoring.TraceCacheGetMatches(ctx, string(data))
	defer finish()

	return c.underlying.GetMatches(newCtx, data)
}

// SaveMatches starts a DataDog trace span and delegates to the underlying cache,
// forwarding the span-bearing context so the token-scoped cache operation
// nests under the span.
func (c *CacheStoreTracer) SaveMatches(ctx context.Context, data []byte, results map[pattern.Entity]bool) error {
	newCtx, finish := monitoring.TraceCacheSaveMatches(ctx, string(data))
	defer finish()

	return c.underlying.SaveMatches(newCtx, data, results)
}
