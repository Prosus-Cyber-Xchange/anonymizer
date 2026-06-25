package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

// EnvConfig holds all environment configuration for the application.
type EnvConfig struct {
	LogLevel    string `env:"LOG_LEVEL" envDefault:"INFO"`
	ServiceName string `env:"SERVICE_NAME" envDefault:""`

	Privacy struct {
		Cache                  bool          `env:"CACHE_ENABLED" envDefault:"false"`
		CacheTTL               time.Duration `env:"CACHE_TTL" envDefault:"1h"`
		RedisCacheAddr         string        `env:"CACHE_REDIS_ADDR" envDefault:""`
		RedisDisableCluster    bool          `env:"CACHE_REDIS_DISABLE_CLUSTER" envDefault:"false"`

		RedisDialTimeout  time.Duration `env:"CACHE_REDIS_DIAL_TIMEOUT" envDefault:"0"`
		RedisReadTimeout  time.Duration `env:"CACHE_REDIS_READ_TIMEOUT" envDefault:"0"`
		RedisWriteTimeout time.Duration `env:"CACHE_REDIS_WRITE_TIMEOUT" envDefault:"0"`
		RedisPoolSize     int           `env:"CACHE_REDIS_POOL_SIZE" envDefault:"0"`
		RedisMinIdleConns int           `env:"CACHE_REDIS_MIN_IDLE_CONNS" envDefault:"0"`

		CacheMetrics bool `env:"CACHE_METRICS" envDefault:"true"`

		DisableInMemoryCache bool `env:"CACHE_DISABLE_IN_MEMORY" envDefault:"false"`

		RedisToken string `env:"REDIS_CACHE_TOKEN" envDefault:""`

		// Concurrency settings
		ConcurrencyEnabled                 bool          `env:"CONCURRENCY_ENABLED" envDefault:"false"`
		ConcurrencyTokenProcessing         bool          `env:"CONCURRENCY_TOKEN_PROCESSING" envDefault:"false"`
		ConcurrencyRuleProcessing          bool          `env:"CONCURRENCY_RULE_PROCESSING" envDefault:"false"`
		ConcurrencyRuleRunnerPoolSize      int           `env:"CONCURRENCY_RULE_RUNNER_POOL_SIZE" envDefault:"0"`
		ConcurrencyTokenPoolSize           int           `env:"CONCURRENCY_TOKEN_POOL_SIZE" envDefault:"0"`
		ConcurrencyMaxGoroutineIdleTimeout time.Duration `env:"CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT" envDefault:"10s"`
	} `envPrefix:"PRIVACY_"`

	MaxBatchSize             int  `env:"MAX_BATCH_SIZE" envDefault:"100"`
	PatternMonitoringEnabled bool `env:"PATTERN_MONITORING_ENABLED" envDefault:"false"`

	// OTel configuration
	OTel struct {
		Enabled      bool   `env:"ENABLED" envDefault:"false"`
		ExporterAddr string `env:"EXPORTER_ADDR" envDefault:"localhost:4317"`
	} `envPrefix:"OTEL_"`

	Server ServerConfig
}

// LoadEnv loads environment variables into EnvConfig struct.
func LoadEnv() (EnvConfig, error) {
	var cfg EnvConfig
	if err := env.Parse(&cfg); err != nil {
		return EnvConfig{}, err
	}
	return cfg, nil
}

// MustLoadEnv loads environment variables and panics on error.
func MustLoadEnv() EnvConfig {
	cfg, err := LoadEnv()
	if err != nil {
		panic(err)
	}
	return cfg
}
