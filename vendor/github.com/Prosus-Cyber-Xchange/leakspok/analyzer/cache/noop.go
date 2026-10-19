package cache

import (
	"context"

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

// GetMatches always returns an empty map: every entity is a cache miss.
func (r NoopRuleMatchingCache) GetMatches(_ context.Context, _ []byte) (map[pattern.Entity]bool, error) {
	return map[pattern.Entity]bool{}, nil
}

// SaveMatches always returns nil, silently discarding the batch.
func (r NoopRuleMatchingCache) SaveMatches(_ context.Context, _ []byte, _ map[pattern.Entity]bool) error {
	return nil
}
