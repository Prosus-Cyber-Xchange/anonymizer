package cache

import (
	"errors"

	"github.com/valkey-io/valkey-go"
)

// ErrCacheNotFound indicates a cache miss (key not found).
var ErrCacheNotFound = errors.New("cache: key not found")

// IsCacheNotFoundError checks if an error represents a cache miss (key not found).
// It detects the sentinel ErrCacheNotFound and valkey nil responses.
func IsCacheNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrCacheNotFound) || valkey.IsValkeyNil(err)
}
