package cache

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// CacheStore defines the interface for caching rule matching results.
// The token-scoped methods store one hash per token: GetMatches reads every
// entity cached for a token in one command, and SaveMatches persists a batch
// of results for a token in one command.
type CacheStore interface {
	// GetMatches retrieves every rule-matching result stored for the token's
	// hash with a single command. The returned map holds exactly the entities
	// present in the hash; an entity absent from the map is a cache miss. An
	// empty map with a nil error means nothing is cached for the token.
	GetMatches(ctx context.Context, data []byte) (map[pattern.Entity]bool, error)

	// SaveMatches persists a batch of rule-matching results for the token's
	// hash in a single command. The caller owns the persistence policy; the
	// store persists every provided entity as given.
	SaveMatches(ctx context.Context, data []byte, results map[pattern.Entity]bool) error
}
