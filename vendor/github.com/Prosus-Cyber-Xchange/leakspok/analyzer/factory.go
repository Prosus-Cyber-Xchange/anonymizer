package analyzer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	analyzercache "github.com/Prosus-Cyber-Xchange/leakspok/analyzer/cache"
	"github.com/panjf2000/ants/v2"
)

func poolOptions(opts ConcurrencyOptions) []ants.Option {
	var out []ants.Option
	if opts.MaxGoroutineIdleTimeout > 0 {
		out = append(out, ants.WithExpiryDuration(opts.MaxGoroutineIdleTimeout))
	}
	return out
}

// CacheOptions configures the rule matching cache behavior.
type CacheOptions struct {
	// Enabled controls whether caching is active.
	Enabled bool
	// Backend selects the cache implementation. Default: gocache.
	Backend string
	// TTL is the time-to-live for cached entries. When 0, entries never expire.
	TTL time.Duration
	// MemorySize is the maximum size of the in-memory cache in bytes.
	// When 0, a default of 10MB (10 * 1024 * 1024) is used.
	// Set to -1 to disable in-memory caching (or client-side caching for the valkey backend).
	MemorySize int
	// RedisAddr is the Redis/Valkey node address(es) in the format "host:port"
	// or "host1:port1,host2:port2,..." for multiple nodes.
	// When empty, the Redis/Valkey backend is disabled.
	RedisAddr string
	// RedisUsername is the ACL username for Redis 6.0+ authentication. Optional.
	RedisUsername string
	// RedisPassword is the password for Redis authentication. Optional.
	RedisPassword string
	// RedisDialTimeout is the timeout for establishing new Redis connections. Optional.
	RedisDialTimeout time.Duration
	// RedisReadTimeout is the timeout for Redis socket reads. Optional.
	RedisReadTimeout time.Duration
	// RedisWriteTimeout is the timeout for Redis socket writes. Optional.
	RedisWriteTimeout time.Duration
	// RedisPoolSize is the maximum number of Redis socket connections per CPU. Optional.
	RedisPoolSize int
	// RedisMinIdleConns is the minimum number of idle Redis connections to maintain. Optional.
	RedisMinIdleConns int
	// RedisInsecureSkipVerify controls whether TLS certificate verification is skipped.
	// Only applies when TLS is enabled (i.e. RedisPassword is set). Default: false.
	RedisInsecureSkipVerify bool
	// RedisPingOnConnect controls whether a Ping is sent to Redis to verify connectivity
	// when the client is first created. Default: false.
	RedisPingOnConnect bool
	// Metric configures Prometheus metrics (gocache backend only).
	Metric analyzercache.RuleMatchingCacheMetricOptions
	// TracingEnabled enables DataDog APM tracing for cache store operations.
	// Each GetMatch and SaveMatch call emits a DataDog span. Requires Enabled=true.
	TracingEnabled bool
}

// ConcurrencyOptions configures concurrent execution behavior for both the rules runner
// and token-level processing in ByteAnalyzer.
type ConcurrencyOptions struct {
	// Enabled is the master switch. When false, all concurrent processing is disabled
	// regardless of other flags.
	Enabled bool
	// ConcurrentTokenProcessing enables parallel token evaluation in ByteAnalyzer.Anonymize.
	// When true, each token is dispatched to a goroutine instead of processed sequentially.
	// Results are sorted by token start position before applying anonymization, ensuring
	// byte-for-byte identical output to the serial path. Requires Enabled=true.
	ConcurrentTokenProcessing bool
	// ConcurrentRuleProcessing enables parallel rule evaluation via ConcurrentRulesRunner.
	// Requires Enabled=true.
	ConcurrentRuleProcessing bool

	// RuleRunnerPoolSize is the size of the goroutine pool used by ConcurrentRulesRunner.
	// Must be > 0 when Enabled=true and ConcurrentRuleProcessing=true.
	RuleRunnerPoolSize int
	// TokenPoolSize is the size of the goroutine pool used by ByteAnalyzer for concurrent
	// token processing. Must be > 0 when Enabled=true and ConcurrentTokenProcessing=true.
	TokenPoolSize int
	// MaxGoroutineIdleTimeout is the duration after which idle goroutines in either pool
	// are reclaimed. When zero, ants uses its default (1 second).
	MaxGoroutineIdleTimeout time.Duration
}

// RunnerOptions configures the cache and concurrency behavior of ByteAnalyzer
// and StringAnalyzer instances created via MakeByteAnalyzer / MakeStringAnalyzer.
type RunnerOptions struct {
	Cache       CacheOptions
	Concurrency ConcurrencyOptions
}

func buildCacheStore(ctx context.Context, options CacheOptions) (analyzercache.CacheStore, error) {
	if !options.Enabled {
		return analyzercache.NewNoopRuleMatchingCache(), nil
	}

	c, err := analyzercache.NewCacheStore(ctx, analyzercache.RuleMatchingCacheOptions{
		Backend:  analyzercache.Backend(options.Backend),
		CacheTTL: options.TTL,
		Memory: analyzercache.MemoryOptions{
			Enabled:   options.MemorySize != -1,
			CacheSize: options.MemorySize,
		},
		Redis: analyzercache.RedisOptions{
			Enabled:            options.RedisAddr != "",
			Addr:               options.RedisAddr,
			Username:           options.RedisUsername,
			Password:           options.RedisPassword,
			DialTimeout:        options.RedisDialTimeout,
			ReadTimeout:        options.RedisReadTimeout,
			WriteTimeout:       options.RedisWriteTimeout,
			PoolSize:           options.RedisPoolSize,
			MinIdleConns:       options.RedisMinIdleConns,
			InsecureSkipVerify: options.RedisInsecureSkipVerify,
			PingOnConnect:      options.RedisPingOnConnect,
		},
		Metrics: options.Metric,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	if options.TracingEnabled {
		return analyzercache.NewCacheStoreTracer(c), nil
	}

	return c, nil
}

// createRunnerPool creates a goroutine pool for concurrent rule processing.
// Assumes concurrency is enabled and ConcurrentRuleProcessing is true;
// pool size validation is the caller's responsibility.
func createRunnerPool(concurrency ConcurrencyOptions, logger *slog.Logger) (WorkerPool, error) {
	pool, err := NewAntsWorkerPool(concurrency.RuleRunnerPoolSize, logger, poolOptions(concurrency)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create rule runner pool: %w", err)
	}
	return pool, nil
}

// createTokenPool creates a goroutine pool for concurrent token processing.
// Assumes concurrency is enabled and ConcurrentTokenProcessing is true;
// pool size validation is the caller's responsibility.
func createTokenPool(concurrency ConcurrencyOptions, logger *slog.Logger) (WorkerPool, error) {
	pool, err := NewAntsWorkerPool(concurrency.TokenPoolSize, logger, poolOptions(concurrency)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create token pool: %w", err)
	}
	return pool, nil
}

// MakeByteAnalyzer creates a ByteAnalyzer configured with the given cache and
// concurrency options. It builds the cache store, goroutine pools, and selects
// the appropriate rule runner based on the provided settings.
//
// Returns an error if the cache or pool initialization fails, or if required
// pool sizes are invalid.
func MakeByteAnalyzer(ctx context.Context, logger *slog.Logger, options RunnerOptions) (ByteAnalyzer, error) {
	cache, err := buildCacheStore(ctx, options.Cache)
	if err != nil {
		return ByteAnalyzer{}, err
	}

	concurrency := options.Concurrency

	var runnerPool WorkerPool
	if concurrency.Enabled && concurrency.ConcurrentRuleProcessing {
		if concurrency.RuleRunnerPoolSize <= 0 {
			return ByteAnalyzer{}, errors.New("RuleRunnerPoolSize must be > 0 when ConcurrentRuleProcessing is enabled")
		}
		runnerPool, err = createRunnerPool(concurrency, logger)
		if err != nil {
			return ByteAnalyzer{}, err
		}
	}

	var tokenPool WorkerPool
	if concurrency.Enabled && concurrency.ConcurrentTokenProcessing {
		if concurrency.TokenPoolSize <= 0 {
			return ByteAnalyzer{}, errors.New("TokenPoolSize must be > 0 when ConcurrentTokenProcessing is enabled")
		}
		tokenPool, err = createTokenPool(concurrency, logger)
		if err != nil {
			return ByteAnalyzer{}, err
		}
	}

	// ConcurrentRuleProcessing and ConcurrentTokenProcessing independently control
	// their own pool. Only one level should be active at a time to avoid deadlocks:
	// - ConcurrentTokenProcessing=true: tokenPool handles token-level parallelism, rules run serially.
	// - ConcurrentRuleProcessing=true: runnerPool handles rule-level parallelism, tokens run serially.
	var runner RuleRunner
	if concurrency.Enabled && concurrency.ConcurrentRuleProcessing {
		runner = NewConcurrentRulesRunner(logger, options, cache, runnerPool)
	} else {
		runner = NewSerialRulesRuner(logger, cache)
	}

	ba := NewByteAnalyzer(logger, runner)
	if concurrency.Enabled && concurrency.ConcurrentTokenProcessing {
		ba.pool = tokenPool
	}

	return ba, nil
}

// MakeStringAnalyzer creates a StringAnalyzer configured with the given cache and
// concurrency options. It builds the cache store, selects the appropriate rule
// runner, and returns a StringAnalyzer that operates on strings instead of byte slices.
//
// Returns an error if the cache initialization or pool creation fails.
func MakeStringAnalyzer(ctx context.Context, logger *slog.Logger, options RunnerOptions) (StringAnalyzer, error) {
	cache, err := buildCacheStore(ctx, options.Cache)
	if err != nil {
		return StringAnalyzer{}, err
	}

	concurrency := options.Concurrency

	var runnerPool WorkerPool
	if concurrency.Enabled && concurrency.ConcurrentRuleProcessing {
		if concurrency.RuleRunnerPoolSize <= 0 {
			return StringAnalyzer{}, errors.New("RuleRunnerPoolSize must be > 0 when ConcurrentRuleProcessing is enabled")
		}
		runnerPool, err = createRunnerPool(concurrency, logger)
		if err != nil {
			return StringAnalyzer{}, err
		}
	}

	var runner RuleRunner
	if concurrency.Enabled && concurrency.ConcurrentRuleProcessing {
		runner = NewConcurrentRulesRunner(logger, options, cache, runnerPool)
	} else {
		runner = NewSerialRulesRuner(logger, cache)
	}

	return NewStringAnalyzer(logger, runner), nil
}
