package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRedis(t *testing.T) config.EnvConfig {
	t.Helper()

	ctx := context.Background()

	redisContainer, err := tcredis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
	)
	require.NoError(t, err, "failed to start redis container")

	t.Cleanup(func() {
		require.NoError(t, redisContainer.Terminate(ctx), "failed to terminate redis container")
	})

	redisAddr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err, "failed to get redis connection string")
	redisAddr = strings.TrimPrefix(redisAddr, "redis://")

	return config.EnvConfig{
		LogLevel:    "INFO",
		ServiceName: "",
		Privacy: struct {
			Cache                  bool          `env:"CACHE_ENABLED" envDefault:"false"`
			CacheTTL               time.Duration `env:"CACHE_TTL" envDefault:"1h"`
			RedisCacheAddr         string        `env:"CACHE_REDIS_ADDR" envDefault:""`
			RedisDisableCluster    bool          `env:"CACHE_REDIS_DISABLE_CLUSTER" envDefault:"false"`
			RedisDialTimeout       time.Duration `env:"CACHE_REDIS_DIAL_TIMEOUT" envDefault:"0"`
			RedisReadTimeout       time.Duration `env:"CACHE_REDIS_READ_TIMEOUT" envDefault:"0"`
			RedisWriteTimeout      time.Duration `env:"CACHE_REDIS_WRITE_TIMEOUT" envDefault:"0"`
			RedisPoolSize          int           `env:"CACHE_REDIS_POOL_SIZE" envDefault:"0"`
			RedisMinIdleConns      int           `env:"CACHE_REDIS_MIN_IDLE_CONNS" envDefault:"0"`
			CacheMetrics           bool          `env:"CACHE_METRICS" envDefault:"true"`
			DisableInMemoryCache   bool          `env:"CACHE_DISABLE_IN_MEMORY" envDefault:"false"`
			RedisToken             string        `env:"REDIS_CACHE_TOKEN" envDefault:""`
			ConcurrencyEnabled                 bool          `env:"CONCURRENCY_ENABLED" envDefault:"false"`
			ConcurrencyTokenProcessing         bool          `env:"CONCURRENCY_TOKEN_PROCESSING" envDefault:"false"`
			ConcurrencyRuleProcessing          bool          `env:"CONCURRENCY_RULE_PROCESSING" envDefault:"false"`
			ConcurrencyRuleRunnerPoolSize      int           `env:"CONCURRENCY_RULE_RUNNER_POOL_SIZE" envDefault:"0"`
			ConcurrencyTokenPoolSize           int           `env:"CONCURRENCY_TOKEN_POOL_SIZE" envDefault:"0"`
			ConcurrencyMaxGoroutineIdleTimeout time.Duration `env:"CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT" envDefault:"10s"`
		}{
			Cache:               true,
			CacheTTL:            5 * time.Minute,
			RedisCacheAddr:       redisAddr,
			RedisDisableCluster: true,
		},
		MaxBatchSize:             100,
		PatternMonitoringEnabled: false,
		Server:                   config.ServerConfig{Port: 0, Host: "0.0.0.0", GracefulShutdownTimeout: 30 * time.Second},
	}
}

func TestNewFromConfig_NoPlugins(t *testing.T) {
	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)
	require.NotNil(t, app)

	h := app.Handler()
	assert.NotNil(t, h)
}

func TestNewFromConfig_DefaultConfig(t *testing.T) {
	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)
	require.NotNil(t, app)

	h := app.Handler()
	assert.NotNil(t, h)
}

func TestHandler_CEEndpointsWork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)

	h := app.Handler()

	body := `{"text":"Contact user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "REDACTED")
	assert.NotContains(t, rec.Body.String(), "user@example.com")
}

func TestHandler_BatchEndpointWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)

	h := app.Handler()

	body := `[{"text":"user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "REDACTED")
}

func TestHandler_HealthEndpointWorks(t *testing.T) {
	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)

	h := app.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_UnknownRouteReturns404(t *testing.T) {
	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)

	h := app.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unknown", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

type mockMiddlewarePlugin struct {
	called bool
}

func (m *mockMiddlewarePlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
	m.called = true
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Plugin-Active", "true")
			next.ServeHTTP(w, r)
		})
	}
}

func TestNewFromConfig_WithMiddlewarePlugin(t *testing.T) {
	plugin := &mockMiddlewarePlugin{}
	cfg := setupRedis(t)

	app, err := server.NewFromConfig(context.Background(),
		server.WithEnv(cfg),
		server.WithPlugin(plugin),
	)
	require.NoError(t, err)

	h := app.Handler()
	assert.True(t, plugin.called)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Result().Header.Get("X-Plugin-Active"))
}

func TestHandler_PluginInjectsContextRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	plugin := &contextRulesPlugin{}
	cfg := setupRedis(t)

	app, err := server.NewFromConfig(context.Background(),
		server.WithEnv(cfg),
		server.WithPlugin(plugin),
	)
	require.NoError(t, err)

	h := app.Handler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("Contact john@example.com"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "john@example.com")
}

type contextRulesPlugin struct{}

func (p *contextRulesPlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<PLUGIN_REDACTED>"}},
		},
	}
	rules, _ := privacy.NewRuleBuilder(settings).Build()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := server.WithRules(r.Context(), rules)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestListenAndServe_RespectsContextCancellation(t *testing.T) {
	cfg := setupRedis(t)
	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = app.ListenAndServe(ctx)
	_ = err
}

func TestNewFromConfig_ConcurrencyConfigWired(t *testing.T) {
	cfg := setupRedis(t)
	cfg.Privacy.ConcurrencyEnabled = true
	cfg.Privacy.ConcurrencyTokenProcessing = true
	cfg.Privacy.ConcurrencyRuleProcessing = true
	cfg.Privacy.ConcurrencyRuleRunnerPoolSize = 4
	cfg.Privacy.ConcurrencyTokenPoolSize = 8

	app, err := server.NewFromConfig(context.Background(), server.WithEnv(cfg))
	require.NoError(t, err)
	require.NotNil(t, app)

	h := app.Handler()
	assert.NotNil(t, h)
}