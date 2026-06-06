package cache

import (
	"context"
	"fmt"
)

// NewCacheStore creates a CacheStore based on the Backend field of the provided options.
// When Backend is empty it defaults to BackendGocache for backwards compatibility.
func NewCacheStore(ctx context.Context, options RuleMatchingCacheOptions) (CacheStore, error) {
	switch options.Backend {
	case BackendValkey:
		return NewValkeyRuleMatchingCache(ctx, options)
	default: // BackendGocache or empty
		c, err := NewRuleMatchingCache(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create gocache store: %w", err)
		}

		return c, nil
	}
}
