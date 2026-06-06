package cache

import (
	"context"
)

// NewCacheStore creates a CacheStore using valkey-go.
func NewCacheStore(ctx context.Context, options RuleMatchingCacheOptions) (CacheStore, error) {
	return NewRuleMatchingCache(ctx, options)
}
