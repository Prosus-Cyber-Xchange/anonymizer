package cache

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// CacheStore defines the interface for caching rule matching results.
// The key is composed of the pattern entity name and the input data, so the
// cache only needs the entity identifier — not the full Rule struct.
type CacheStore interface {
	GetMatch(ctx context.Context, entity pattern.Entity, data []byte) (bool, error)
	SaveMatch(ctx context.Context, entity pattern.Entity, data []byte, matched bool) error
}
