package cache

import (
	"errors"

	"github.com/coocood/freecache"
	"github.com/eko/gocache/lib/v4/store"
	"github.com/redis/go-redis/v9"
)

// IsCacheNotFoundError checks if an error represents a cache miss (key not found).
// It detects cache miss errors from the gocache library, freecache, and Redis backends.
// Returns true if the error is a "not found" error, false otherwise.
func IsCacheNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, store.NotFound{}):
		return true
	case errors.Is(err, freecache.ErrNotFound):
		return true
	case errors.Is(err, redis.Nil):
		return true
	default:
		return false
	}
}
