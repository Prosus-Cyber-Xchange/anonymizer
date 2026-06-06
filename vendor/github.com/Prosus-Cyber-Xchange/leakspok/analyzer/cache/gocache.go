package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	gocache "github.com/eko/gocache/lib/v4/cache"
	"github.com/eko/gocache/lib/v4/marshaler"
	"github.com/eko/gocache/lib/v4/metrics"
	"github.com/eko/gocache/lib/v4/store"
	gocachestore "github.com/eko/gocache/store/freecache/v4"
	redisstore "github.com/eko/gocache/store/redis/v4"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/redis/go-redis/v9"
)

// RuleMatchingCache caches rule matching results using gocache with optional
// freecache (in-memory) and Redis backends.
//
// # Design Notes
//
// Boolean values are encoded as single-byte slices:
//   - false → []byte{0}
//   - true  → []byte{1}
//
// # Key Generation
//
// Cache keys are constructed as "entity:data" where entity is the pattern entity name
// and data is the input bytes. Keys are generated using a sync.Pool to minimize
// allocations for frequently accessed cache keys.
type RuleMatchingCache struct {
	cache      *marshaler.Marshaler
	keyBufPool sync.Pool
	cacheTTL   time.Duration
}

func buildRedisClient(ctx context.Context, options RedisOptions) (redis.Cmdable, error) {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: options.InsecureSkipVerify, //nolint:gosec // Controlled by caller; disabled by default
	}

	var redisClient redis.Cmdable
	if options.DisableClusterMode {
		redisClientOpts := &redis.Options{
			Addr:         options.Addr,
			Username:     options.Username,
			Password:     options.Password,
			DialTimeout:  options.DialTimeout,
			ReadTimeout:  options.ReadTimeout,
			WriteTimeout: options.WriteTimeout,
			PoolSize:     options.PoolSize,
			MinIdleConns: options.MinIdleConns,
		}
		if options.Password != "" {
			redisClientOpts.TLSConfig = tlsConfig
		}
		redisClient = redis.NewClient(redisClientOpts)
	} else {
		addrs := strings.Split(options.Addr, ",")
		redisClusterClientOpts := &redis.ClusterOptions{
			Addrs:        addrs,
			Username:     options.Username,
			Password:     options.Password,
			DialTimeout:  options.DialTimeout,
			ReadTimeout:  options.ReadTimeout,
			WriteTimeout: options.WriteTimeout,
			PoolSize:     options.PoolSize,
			MinIdleConns: options.MinIdleConns,
		}
		if options.Password != "" {
			redisClusterClientOpts.TLSConfig = tlsConfig
		}
		redisClient = redis.NewClusterClient(redisClusterClientOpts)
	}

	if options.PingOnConnect {
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("redis connection check failed: %w", err)
		}
	}

	return redisClient, nil
}

// NewRuleMatchingCache creates a new instance of RuleMatchingCache.
// It initializes cache backends based on configuration options.
// Supports chaining multiple backends (memory, Redis) for multi-tier caching.
// If no backends are explicitly enabled, memory cache is enabled by default.
func NewRuleMatchingCache(ctx context.Context, options RuleMatchingCacheOptions) (*RuleMatchingCache, error) {
	var stores []gocache.SetterCacheInterface[any]

	memoryEnabled := options.Memory.Enabled || (!options.Memory.Enabled && !options.Redis.Enabled)

	if memoryEnabled {
		cacheSize := options.Memory.CacheSize
		if cacheSize <= 0 {
			cacheSize = 10 * 1024 * 1024 // 10MB default
		}

		freecacheClient := freecache.NewCache(cacheSize)
		cacheStore := gocachestore.NewFreecache(freecacheClient)
		stores = append(stores, gocache.New[any](cacheStore))
	}

	if options.Redis.Enabled {
		redisClient, err := buildRedisClient(ctx, options.Redis)
		if err != nil {
			return nil, err
		}

		redisStore := redisstore.NewRedis(redisClient)
		stores = append(stores, gocache.New[any](redisStore))
	}

	var baseCache gocache.CacheInterface[any]
	switch {
	case len(stores) > 1:
		baseCache = gocache.NewChain[any](stores...)
	case len(stores) == 1:
		baseCache = stores[0]
	default:
		baseCache = disabledCache[any]{}
	}

	var cache gocache.CacheInterface[any]
	if options.Metrics.Enabled {
		promMetrics := metrics.NewPrometheus(
			options.Metrics.ServiceName+"_leakspok",
			metrics.WithRegisterer(options.Metrics.Registry),
		)

		cache = gocache.NewMetric[any](
			promMetrics,
			baseCache,
		)
	} else {
		cache = baseCache
	}

	m := marshaler.New(cache)

	return &RuleMatchingCache{
		cache:    m,
		cacheTTL: options.CacheTTL,
	}, nil
}

// GetMatch retrieves a cached rule matching result for the given entity and data.
func (r *RuleMatchingCache) GetMatch(ctx context.Context, entity pattern.Entity, data []byte) (bool, error) {
	key := r.key(entity, data)

	var cached bool

	_, err := r.cache.Get(ctx, key, &cached)
	if err != nil {
		return false, err
	}

	return cached, nil
}

// SaveMatch caches a rule matching result with the optional TTL.
func (r *RuleMatchingCache) SaveMatch(ctx context.Context, entity pattern.Entity, data []byte, matched bool) error {
	key := r.key(entity, data)

	var opts []store.Option
	if r.cacheTTL > 0 {
		opts = append(opts, store.WithExpiration(r.cacheTTL))
	}

	return r.cache.Set(ctx, key, matched, opts...)
}

// key generates a cache key from a pattern entity and data bytes using the format "entity:data".
// It uses a sync.Pool to minimize allocations.
func (r *RuleMatchingCache) key(x pattern.Entity, y []byte) string {
	n := len(x) + 1 + len(y)

	var buf []byte

	v := r.keyBufPool.Get()
	if v != nil {
		//nolint:forcetypeassert,errcheck // sync.Pool stores []byte, type assertion is safe
		buf = v.([]byte)
	}
	if cap(buf) < n {
		buf = make([]byte, n)
	}
	buf = buf[:n]

	i := copy(buf, x)
	buf[i] = ':'
	copy(buf[i+1:], y)

	s := string(buf)

	//nolint:staticcheck // SA6002: using non-pointer intentionally for buffer reuse
	r.keyBufPool.Put(buf)

	return s
}

// disabledCache is a no-operation cache implementation for gocache.CacheInterface.
type disabledCache[T any] struct{}

func (d disabledCache[T]) Get(_ context.Context, _ any) (T, error) {
	var empty T
	return empty, store.NotFound{}
}

func (d disabledCache[T]) Set(_ context.Context, _ any, _ T, _ ...store.Option) error {
	return nil
}

func (d disabledCache[T]) Delete(_ context.Context, _ any) error {
	return nil
}

func (d disabledCache[T]) Invalidate(_ context.Context, _ ...store.InvalidateOption) error {
	return nil
}

func (d disabledCache[T]) Clear(_ context.Context) error {
	return nil
}

func (d disabledCache[T]) GetType() string {
	return "DisabledCache"
}
