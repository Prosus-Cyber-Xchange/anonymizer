package cache

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/valkey-io/valkey-go"
)

// RuleMatchingCache caches rule matching results using valkey-go.
//
// It benefits from two valkey-go features with zero extra code:
//   - Auto pipelining: concurrent Do/DoCache calls from different goroutines are
//     automatically batched into a single network round-trip.
//   - Client-side caching: DoCache stores results in local process memory and the
//     server pushes invalidation messages when keys change, eliminating round-trips
//     for hot keys on subsequent requests.
//
// Client-side caching is disabled when RuleMatchingCacheOptions.DisableInMemoryCache is true.
type RuleMatchingCache struct {
	client     valkey.Client
	cacheTTL   time.Duration
	disableCSC bool
	keyBufPool sync.Pool
}

// NewRuleMatchingCache creates a RuleMatchingCache from the provided options.
func NewRuleMatchingCache(ctx context.Context, options RuleMatchingCacheOptions) (*RuleMatchingCache, error) {
	r := options.Redis

	opt := valkey.ClientOption{
		InitAddress:       strings.Split(r.Addr, ","),
		Username:          r.Username,
		Password:          r.Password,
		ForceSingleClient: r.DisableClusterMode,
		DisableCache:      options.DisableInMemoryCache,
		Dialer: net.Dialer{
			Timeout: r.DialTimeout,
		},
	}

	if r.WriteTimeout > 0 {
		opt.ConnWriteTimeout = r.WriteTimeout
	}

	if r.PoolSize > 0 {
		opt.BlockingPoolSize = r.PoolSize
	}

	if r.Password != "" {
		opt.TLSConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: r.InsecureSkipVerify, //nolint:gosec // Controlled by caller; disabled by default
		}
	}

	client, err := valkey.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to create valkey client: %w", err)
	}

	if r.PingOnConnect {
		if pingErr := client.Do(ctx, client.B().Ping().Build()).Error(); pingErr != nil {
			client.Close()
			return nil, fmt.Errorf("valkey connection check failed: %w", pingErr)
		}
	}

	return &RuleMatchingCache{
		client:     client,
		cacheTTL:   options.CacheTTL,
		disableCSC: options.DisableInMemoryCache,
	}, nil
}

// GetMatch retrieves a cached rule matching result.
// Uses DoCache for client-side caching unless disabled; falls back to Do otherwise.
// Auto pipelining coalesces concurrent calls into a single network round-trip.
func (r *RuleMatchingCache) GetMatch(ctx context.Context, entity pattern.Entity, data []byte) (bool, error) {
	key := r.key(entity, data)

	var result valkey.ValkeyResult

	if r.disableCSC || r.cacheTTL == 0 {
		result = r.client.Do(ctx, r.client.B().Get().Key(key).Build())
	} else {
		result = r.client.DoCache(ctx, r.client.B().Get().Key(key).Cache(), r.cacheTTL)
	}

	val, err := result.ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return false, ErrCacheNotFound
		}

		return false, err
	}

	return val == "1", nil
}

// SaveMatch caches a rule matching result.
// Auto pipelining coalesces concurrent writes into a single network round-trip.
func (r *RuleMatchingCache) SaveMatch(ctx context.Context, entity pattern.Entity, data []byte, matched bool) error {
	key := r.key(entity, data)

	val := "0"
	if matched {
		val = "1"
	}

	var cmd valkey.Completed
	if r.cacheTTL > 0 {
		cmd = r.client.B().Set().Key(key).Value(val).Px(r.cacheTTL).Build()
	} else {
		cmd = r.client.B().Set().Key(key).Value(val).Build()
	}

	return r.client.Do(ctx, cmd).Error()
}

// key generates a cache key in "entity:data" format, reusing a sync.Pool buffer.
func (r *RuleMatchingCache) key(x pattern.Entity, y []byte) string {
	n := len(x) + 1 + len(y)

	var buf []byte

	if raw := r.keyBufPool.Get(); raw != nil {
		//nolint:forcetypeassert,errcheck // sync.Pool stores []byte, type assertion is safe
		buf = raw.([]byte)
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
