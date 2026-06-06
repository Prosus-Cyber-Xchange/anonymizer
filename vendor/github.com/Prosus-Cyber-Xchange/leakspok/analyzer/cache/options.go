package cache

import (
	"time"
)

// RuleMatchingCacheOptions configures a RuleMatchingCache instance.
type RuleMatchingCacheOptions struct {
	// CacheTTL is the time-to-live for cached entries. When 0, entries never expire.
	CacheTTL time.Duration
	// DisableInMemoryCache disables client-side caching (server-assisted CSC).
	// When false (default), valkey-go uses DoCache for local caching with
	// server-driven invalidation. Set to true to disable and always hit the server.
	DisableInMemoryCache bool
	// Redis configures the Redis/Valkey backend connection.
	Redis RedisOptions
}

// RedisOptions configures the Redis/Valkey cache backend connection.
type RedisOptions struct {
	// Addr is the node address(es) in the format "host:port" for standalone mode,
	// or "host1:port1,host2:port2,..." for cluster mode.
	Addr string
	// DisableClusterMode disables cluster mode and uses a standalone client instead.
	// Default: false (cluster mode enabled).
	DisableClusterMode bool
	// Username is the ACL username for Redis 6.0+ authentication. Optional.
	Username string
	// Password is the password for authentication. Optional.
	Password string
	// DialTimeout is the timeout for establishing new connections. Optional.
	DialTimeout time.Duration
	// ReadTimeout is the timeout for socket reads. Optional.
	ReadTimeout time.Duration
	// WriteTimeout is the timeout for socket writes. Optional.
	WriteTimeout time.Duration
	// PoolSize is the maximum number of socket connections per CPU. Optional.
	PoolSize int
	// MinIdleConns is the minimum number of idle connections to maintain. Optional.
	MinIdleConns int
	// InsecureSkipVerify controls whether TLS certificate verification is skipped.
	// Only applies when TLS is enabled (i.e. Password is set). Default: false.
	InsecureSkipVerify bool
	// PingOnConnect controls whether a Ping is sent to verify connectivity on first connect.
	// Default: false.
	PingOnConnect bool
}
