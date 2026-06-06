package cache

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Backend selects the cache implementation to use.
type Backend string

const (
	// BackendGocache uses the existing gocache + freecache + go-redis implementation.
	// This is the default for backwards compatibility.
	BackendGocache Backend = "gocache"
	// BackendValkey uses valkey-go with auto pipelining and server-assisted client-side caching.
	BackendValkey Backend = "valkey"
)

// RuleMatchingCacheMetricOptions configures Prometheus metrics for the cache.
type RuleMatchingCacheMetricOptions struct {
	Enabled     bool
	Registry    prometheus.Registerer
	ServiceName string
}

// RedisOptions configures the Redis/Valkey cache backend connection.
type RedisOptions struct {
	// Enabled determines whether this backend is used.
	Enabled bool
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

// MemoryOptions configures the in-memory (freecache) cache backend.
// For the valkey backend, CacheSize controls client-side caching:
// set to -1 to disable client-side caching.
type MemoryOptions struct {
	// Enabled determines whether in-memory caching is used (gocache backend only).
	Enabled bool
	// CacheSize is the maximum size of the freecache in bytes (gocache backend).
	// For the valkey backend, set to -1 to disable client-side caching.
	// Default: 10MB (10 * 1024 * 1024).
	CacheSize int
}

// RuleMatchingCacheOptions configures a RuleMatchingCache instance.
type RuleMatchingCacheOptions struct {
	// Backend selects the cache implementation. Default: BackendGocache.
	Backend Backend
	// CacheTTL is the time-to-live for cached entries. When 0, entries never expire.
	CacheTTL time.Duration
	// Memory configures the in-memory cache backend.
	Memory MemoryOptions
	// Redis configures the Redis/Valkey backend connection.
	Redis RedisOptions
	// Metrics configures Prometheus metrics (gocache backend only).
	Metrics RuleMatchingCacheMetricOptions
}
