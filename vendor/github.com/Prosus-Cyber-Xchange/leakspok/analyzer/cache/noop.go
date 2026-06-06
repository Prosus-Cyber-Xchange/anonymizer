package cache

import (
	"context"

	"github.com/eko/gocache/lib/v4/store"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// NoopRuleMatchingCache is a no-operation cache implementation that doesn't store anything.
// It always returns a cache miss, forcing actual matcher evaluation every time.
// Useful for disabling caching while maintaining the same interface.
type NoopRuleMatchingCache struct{}

// NewNoopRuleMatchingCache creates a new instance of NoopRuleMatchingCache.
func NewNoopRuleMatchingCache() NoopRuleMatchingCache {
	return NoopRuleMatchingCache{}
}

// GetMatch always returns false with a store.NotFound error, simulating a cache miss.
func (r NoopRuleMatchingCache) GetMatch(_ context.Context, _ pattern.Entity, _ []byte) (bool, error) {
	return false, store.NotFound{}
}

// SaveMatch always returns nil, silently discarding the match result.
func (r NoopRuleMatchingCache) SaveMatch(_ context.Context, _ pattern.Entity, _ []byte, _ bool) error {
	return nil
}
