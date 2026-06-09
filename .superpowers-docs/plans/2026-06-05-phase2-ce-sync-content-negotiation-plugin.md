# Phase 2: CE Sync, Content Negotiation & Plugin System — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring anonymizer-service-v2-ce to feature parity with the internal v2 service, add text/plain content negotiation, and simplify the plugin system to a single `MiddlewareRegistrar` interface.

**Architecture:** Sequential subphases (2A cleanup → 2B content negotiation → 2C plugin hardening). Each subphase produces a green test suite. The handler uses a codec dispatcher pattern for content negotiation. Plugins inject middleware that can store pre-built rules into request context.

**Tech Stack:** Go 1.25, chi v5, OpenTelemetry (replacing DataDog), slog (replacing foodsec-go-sdk logging), caarlos0/env/v11, leakspok, testify + uber-go/mock

---

## Task 1: Delete Stale Duplicate Packages

**Files:**
- Delete: `internal/api/` (entire directory)
- Delete: `internal/privacy/` (entire directory)
- Delete: `internal/config/` (entire directory)

- [ ] **Step 1: Delete the duplicate directories**

```bash
rm -rf internal/api/ internal/privacy/ internal/config/
```

- [ ] **Step 2: Verify compilation fails (expected — imports still reference deleted packages)**

Run: `go build ./... 2>&1 | head -20`
Expected: Compilation errors in files that imported `internal/config`, `internal/privacy`, or `internal/api`.

- [ ] **Step 3: Fix remaining imports in internal/handler/**

In `internal/handler/handler.go`, the import `anonymizer-service-v2/pkg/config` already exists. No changes needed for that file. Check `internal/handler/router.go` — it does NOT import from `internal/config` or `internal/privacy` (it uses `pkg/privacy`). No changes needed.

Run: `grep -r "internal/config\|internal/privacy\|internal/api" --include="*.go" . | grep -v vendor | grep -v "_test.go"`

Expected: Zero matches (only test files may still reference deleted mocks — we fix those next).

- [ ] **Step 4: Fix test imports if any reference deleted packages**

Run: `grep -r "internal/config\|internal/privacy\|internal/api" --include="*_test.go" . | grep -v vendor`

If any match: update those imports to reference `pkg/config`, `pkg/privacy`, or remove the test if it's a pure duplicate.

- [ ] **Step 5: Verify compilation succeeds**

Run: `go build ./...`
Expected: Success (exit 0)

- [ ] **Step 6: Run tests**

Run: `go test ./...`
Expected: All tests pass (the deleted packages had test files that are now gone — the remaining `internal/handler/`, `pkg/` tests should still pass).

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: delete stale duplicate packages (internal/api, internal/privacy, internal/config)"
```

---

## Task 2: Replace foodsec-go-sdk Error Response Helper

**Files:**
- Create: `internal/handler/errors.go`
- Modify: `internal/handler/handler.go`

- [ ] **Step 1: Write the failing test for respondError**

Create `internal/handler/errors_test.go`:

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondError_WritesJSONError(t *testing.T) {
	rec := httptest.NewRecorder()

	respondError(rec, http.StatusBadRequest, "INVALID_REQUEST", "something went wrong")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"code":"INVALID_REQUEST","error":"something went wrong"}`, rec.Body.String())
}

func TestRespondError_InternalServerError(t *testing.T) {
	rec := httptest.NewRecorder()

	respondError(rec, http.StatusInternalServerError, "INTERNAL_ERROR", "unexpected failure")

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"code":"INTERNAL_ERROR","error":"unexpected failure"}`, rec.Body.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/ -run TestRespondError -v`
Expected: FAIL — `respondError` undefined

- [ ] **Step 3: Implement respondError**

Create `internal/handler/errors.go`:

```go
package handler

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Code  string `json:"code"`
	Error string `json:"error"`
}

func respondError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(errorResponse{Code: code, Error: message})
}

func respondJSON(w http.ResponseWriter, statusCode int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(body)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/ -run TestRespondError -v`
Expected: PASS

- [ ] **Step 5: Replace all foodsechttp usages in handler.go**

In `internal/handler/handler.go`:
- Remove import: `foodsechttp "code.ifoodcorp.com.br/ifood/security/libs/go/foodsec-go-sdk/http"`
- Remove field: `restEncoder foodsechttp.RESTEncoder` from `Handler` struct
- Remove from `NewHandlerWithMetrics`: `restEncoder: foodsechttp.NewRestEncoder(logger),`
- Replace all `h.restEncoder.RespondError(w, ...)` calls with `respondError(w, statusCode, code, message)`
- Replace all `h.restEncoder.Respond(w, ...)` calls with `respondJSON(w, http.StatusOK, body)`

Each replacement mapping:
```
h.restEncoder.RespondError(w, foodsechttp.WithErrorStatusCode(http.StatusBadRequest), foodsechttp.WithErrorCode("INVALID_REQUEST"), foodsechttp.WithError(err))
→ respondError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())

h.restEncoder.RespondError(w, foodsechttp.WithErrorStatusCode(http.StatusBadRequest), foodsechttp.WithErrorCode("INVALID_SETTINGS"), foodsechttp.WithError(err))
→ respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())

h.restEncoder.RespondError(w, foodsechttp.WithErrorStatusCode(http.StatusInternalServerError), foodsechttp.WithErrorCode("ANONYMIZATION_FAILED"), foodsechttp.WithError(err))
→ respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())

h.restEncoder.RespondError(w, foodsechttp.WithErrorStatusCode(http.StatusBadRequest), foodsechttp.WithErrorCode("BATCH_SIZE_EXCEEDED"), foodsechttp.WithErrorMessage(msg))
→ respondError(w, http.StatusBadRequest, "BATCH_SIZE_EXCEEDED", msg)

h.restEncoder.Respond(w, foodsechttp.WithStatusCode(http.StatusOK), foodsechttp.WithBody(resp))
→ respondJSON(w, http.StatusOK, resp)
```

- [ ] **Step 6: Run all handler tests**

Run: `go test ./internal/handler/ -v`
Expected: All pass. The test assertions check for status codes and body content — the JSON format may differ slightly from foodsec's format. If tests fail, adjust test assertions to match the new `{"code":"...","error":"..."}` format.

- [ ] **Step 7: Commit**

```bash
git add internal/handler/errors.go internal/handler/errors_test.go internal/handler/handler.go
git commit -m "refactor: replace foodsec-go-sdk error responses with stdlib JSON helper"
```

---

## Task 3: Replace foodsec-go-sdk Access Logger and Tracing Middleware

**Files:**
- Create: `internal/handler/access_log.go`
- Modify: `internal/handler/router.go`
- Modify: `internal/monitoring/tracer.go`

- [ ] **Step 1: Write access log middleware test**

Create `internal/handler/access_log_test.go`:

```go
package handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccessLogMiddleware_LogsRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := AccessLog(logger, []string{"/health"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Contains(t, buf.String(), "/api/v1/test")
	assert.Contains(t, buf.String(), "200")
}

func TestAccessLogMiddleware_SkipsHealthRoute(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := AccessLog(logger, []string{"/health"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, buf.String())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/ -run TestAccessLogMiddleware -v`
Expected: FAIL — `AccessLog` undefined

- [ ] **Step 3: Implement access log middleware**

Create `internal/handler/access_log.go`:

```go
package handler

import (
	"log/slog"
	"net/http"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// AccessLog returns a chi-compatible middleware that logs HTTP requests using slog.
// Routes in skipRoutes are not logged.
func AccessLog(logger *slog.Logger, skipRoutes []string) func(http.Handler) http.Handler {
	skip := make(map[string]struct{}, len(skipRoutes))
	for _, route := range skipRoutes {
		skip[route] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(wrapped, r)
			duration := time.Since(start)

			logger.LogAttrs(r.Context(), slog.LevelDebug, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			)
		})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/ -run TestAccessLogMiddleware -v`
Expected: PASS

- [ ] **Step 5: Replace DataDog tracing with OpenTelemetry in monitoring/tracer.go**

Rewrite `internal/monitoring/tracer.go`:

```go
package monitoring

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "anonymizer-service"

// Span wraps an OTel span to provide a uniform interface.
type Span struct {
	span trace.Span
}

// Finish ends the span.
func (s *Span) Finish() {
	if s.span != nil {
		s.span.End()
	}
}

// StartSpan creates and starts a new span in the current trace.
func StartSpan(ctx context.Context, operationName string) (*Span, context.Context) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, operationName)
	return &Span{span: span}, ctx
}

// SetError records an error on the span.
func SetError(span *Span, err error) {
	if span == nil || span.span == nil || err == nil {
		return
	}
	span.span.RecordError(err)
	span.span.SetStatus(codes.Error, err.Error())
}

// SetTag sets a key-value attribute on the span.
func SetTag(span *Span, key string, value interface{}) {
	if span == nil || span.span == nil {
		return
	}
	switch v := value.(type) {
	case string:
		span.span.SetAttributes(attribute.String(key, v))
	case int:
		span.span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.span.SetAttributes(attribute.Int64(key, v))
	case bool:
		span.span.SetAttributes(attribute.Bool(key, v))
	case float64:
		span.span.SetAttributes(attribute.Float64(key, v))
	}
}

// SetTags sets multiple attributes on a span.
func SetTags(span *Span, tags map[string]interface{}) {
	for key, value := range tags {
		SetTag(span, key, value)
	}
}

// FinishWithError finishes a span and tags it with error information.
func FinishWithError(span *Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		SetError(span, err)
	}
	span.Finish()
}
```

- [ ] **Step 6: Update router.go to use stdlib replacements**

Rewrite `internal/handler/router.go`:

```go
package handler

import (
	"anonymizer-service-v2/pkg/privacy"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	leakspokmonitoring "go.ifoodcorp.com.br/leakspok/monitoring"
	"log/slog"
	"net/http"
)

// RouterOptions encapsulates optional configuration for the HTTP router and its handler.
type RouterOptions struct {
	MaxBatchSize            int
	PatternMonitoringEnabled bool
}

func patternMonitoringMiddleware(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := leakspokmonitoring.WithPatternMonitoring(r.Context(), enabled)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// NewRouter creates and configures the HTTP router for the anonymizer API.
func NewRouter(logger *slog.Logger, privacyService *privacy.Service, opts RouterOptions) http.Handler {
	const healthPath = "/health"

	router := chi.NewRouter()

	router.Use(middleware.RealIP)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(middleware.RequestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}
			ctx := r.Context()
			logger.DebugContext(ctx, "request", slog.String("request_id", requestID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Use(middleware.Recoverer)
	router.Use(patternMonitoringMiddleware(opts.PatternMonitoringEnabled))
	router.Use(AccessLog(logger, []string{healthPath}))

	router.Get(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	handler := NewHandler(logger, privacyService, opts.MaxBatchSize)

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/anonymize", handler.Anonymize)
		r.Post("/anonymize/batch", handler.AnonymizeBatch)
	})

	return router
}
```

- [ ] **Step 7: Run all tests**

Run: `go test ./internal/handler/ ./internal/monitoring/ -v`
Expected: All pass (update monitoring test to use new `Span` type if needed).

- [ ] **Step 8: Commit**

```bash
git add internal/handler/access_log.go internal/handler/access_log_test.go internal/handler/router.go internal/monitoring/tracer.go
git commit -m "refactor: replace DataDog tracing and foodsec access log with OTel and slog"
```

---

## Task 4: Replace foodsec-go-sdk in Config and Entrypoint

**Files:**
- Create: `pkg/config/server.go`
- Modify: `pkg/config/env.go`
- Modify: `cmd/server/main.go`
- Modify: `pkg/anonymizer/app.go`

- [ ] **Step 1: Create ServerConfig struct**

Create `pkg/config/server.go`:

```go
package config

import "time"

// ServerConfig holds HTTP server configuration loaded from environment.
type ServerConfig struct {
	Port                    int           `env:"PORT" envDefault:"8080"`
	Host                    string        `env:"HOST" envDefault:"0.0.0.0"`
	GracefulShutdownTimeout time.Duration `env:"GRACEFUL_SHUTDOWN_TIMEOUT" envDefault:"30s"`
}
```

- [ ] **Step 2: Rewrite pkg/config/env.go to remove foodsec-go-sdk**

```go
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
		Cache          bool          `env:"CACHE_ENABLED" envDefault:"false"`
		CacheTTL       time.Duration `env:"CACHE_TTL" envDefault:"1h"`
		CacheSize      int           `env:"CACHE_SIZE" envDefault:"0"`
		RedisCacheAddr string        `env:"REDIS_CACHE_ADDR" envDefault:""`

		RedisDialTimeout  time.Duration `env:"CACHE_REDIS_DIAL_TIMEOUT" envDefault:"0"`
		RedisReadTimeout  time.Duration `env:"CACHE_REDIS_READ_TIMEOUT" envDefault:"0"`
		RedisWriteTimeout time.Duration `env:"CACHE_REDIS_WRITE_TIMEOUT" envDefault:"0"`
		RedisPoolSize     int           `env:"CACHE_REDIS_POOL_SIZE" envDefault:"0"`
		RedisMinIdleConns int           `env:"CACHE_REDIS_MIN_IDLE_CONNS" envDefault:"0"`

		CacheMetrics bool `env:"CACHE_METRICS" envDefault:"true"`

		// Concurrency settings
		ConcurrencyEnabled                 bool          `env:"CONCURRENCY_ENABLED" envDefault:"false"`
		ConcurrencyTokenProcessing         bool          `env:"CONCURRENCY_TOKEN_PROCESSING" envDefault:"false"`
		ConcurrencyRuleProcessing          bool          `env:"CONCURRENCY_RULE_PROCESSING" envDefault:"false"`
		ConcurrencyRuleRunnerPoolSize      int           `env:"CONCURRENCY_RULE_RUNNER_POOL_SIZE" envDefault:"0"`
		ConcurrencyTokenPoolSize           int           `env:"CONCURRENCY_TOKEN_POOL_SIZE" envDefault:"0"`
		ConcurrencyMaxGoroutineIdleTimeout time.Duration `env:"CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT" envDefault:"10s"`
	} `envPrefix:"PRIVACY_"`

	RedisCacheToken string `env:"REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN" envDefault:""`

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
```

- [ ] **Step 3: Rewrite cmd/server/main.go without foodsec-go-sdk**

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"anonymizer-service-v2/pkg/anonymizer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("fatal error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	app, err := anonymizer.NewFromConfig(ctx)
	if err != nil {
		return err
	}

	return app.ListenAndServe(ctx)
}
```

- [ ] **Step 4: Update pkg/anonymizer/app.go — remove foodsec-go-sdk imports and DataDog**

Replace imports and usage:
- Remove `foodsecserver`, `foodseclogging`, `chitrace` imports
- Remove `chitrace.Middleware(...)` from `Handler()`
- Remove `foodseclogging.WithAttribute(...)` — replace with just setting request_id on context or logger
- Remove `foodsecserver.NewAccessLogger(...)` and `foodsecserver.AccessLog(...)` — replace with `handler.AccessLog(a.logger, ...)`
- The `Service` struct field `envConfig config.EnvConfig` — the `Server` field is now `config.ServerConfig`
- In `ListenAndServe`: reference `a.envConfig.Server.Host`, `a.envConfig.Server.Port`, `a.envConfig.Server.GracefulShutdownTimeout` (these names are the same)

Also add logger initialization in `NewFromConfig` if not provided:
```go
if a.logger == nil {
    var level slog.Level
    switch strings.ToUpper(envConfig.LogLevel) {
    case "DEBUG":
        level = slog.LevelDebug
    case "WARN", "WARNING":
        level = slog.LevelWarn
    case "ERROR":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }
    a.logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
```

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: All pass. Fix any remaining references to old imports.

- [ ] **Step 6: Commit**

```bash
git add pkg/config/server.go pkg/config/env.go cmd/server/main.go pkg/anonymizer/app.go
git commit -m "refactor: remove foodsec-go-sdk dependency, use stdlib slog and signal handling"
```

---

## Task 5: Update go.mod — Remove Internal Dependencies, Add OTel

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: Remove foodsec-go-sdk and DataDog from go.mod**

```bash
go mod edit -droprequire code.ifoodcorp.com.br/ifood/security/libs/go/foodsec-go-sdk
go mod edit -droprequire github.com/DataDog/dd-trace-go/contrib/go-chi/chi.v5/v2
go mod edit -droprequire github.com/DataDog/dd-trace-go/v2
```

- [ ] **Step 2: Add OTel dependencies**

```bash
go get go.opentelemetry.io/otel@latest
go get go.opentelemetry.io/otel/sdk@latest
go get go.opentelemetry.io/otel/trace@latest
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp@latest
```

- [ ] **Step 3: Tidy and verify**

```bash
go mod tidy
go build ./...
```

Expected: Builds clean. If there are still indirect references to DataDog via leakspok, that's expected (leakspok's go.mod pulls them in). What matters is CE code no longer directly imports them.

- [ ] **Step 4: Run full test suite**

Run: `go test ./...`
Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum
git commit -m "build: remove foodsec-go-sdk and DataDog deps, add OpenTelemetry"
```

---

## Task 6: Add Concurrency Configuration Wiring

**Files:**
- Modify: `pkg/anonymizer/app.go` (wire concurrency options into `RunnerOptions`)

- [ ] **Step 1: Write test for concurrency config wiring**

Add to `pkg/anonymizer/app_test.go`:

```go
func TestNewFromConfig_ConcurrencyConfigWired(t *testing.T) {
	// Set env vars for concurrency
	t.Setenv("PRIVACY_CONCURRENCY_ENABLED", "true")
	t.Setenv("PRIVACY_CONCURRENCY_TOKEN_PROCESSING", "true")
	t.Setenv("PRIVACY_CONCURRENCY_RULE_PROCESSING", "true")
	t.Setenv("PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE", "4")
	t.Setenv("PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE", "8")

	app, err := anonymizer.NewFromConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, app)

	// Service should initialize without error — verifies config parsing
	h := app.Handler()
	assert.NotNil(t, h)
}
```

- [ ] **Step 2: Run test to verify it passes (config is parsed but wiring may not be complete)**

Run: `go test ./pkg/anonymizer/ -run TestNewFromConfig_ConcurrencyConfigWired -v`

- [ ] **Step 3: Wire concurrency into RunnerOptions in NewFromConfig**

In `pkg/anonymizer/app.go`, update the `MakeByteAnalyzer` call to include concurrency options from `envConfig.Privacy`:

```go
ba, err := analyzer.MakeByteAnalyzer(ctx, a.logger, analyzer.RunnerOptions{
    Cache: analyzer.CacheOptions{
        // ... existing cache options ...
    },
    Concurrency: analyzer.ConcurrencyOptions{
        Enabled:                 envConfig.Privacy.ConcurrencyEnabled,
        TokenProcessing:         envConfig.Privacy.ConcurrencyTokenProcessing,
        RuleProcessing:          envConfig.Privacy.ConcurrencyRuleProcessing,
        RuleRunnerPoolSize:      envConfig.Privacy.ConcurrencyRuleRunnerPoolSize,
        TokenPoolSize:           envConfig.Privacy.ConcurrencyTokenPoolSize,
        MaxGoroutineIdleTimeout: envConfig.Privacy.ConcurrencyMaxGoroutineIdleTimeout,
    },
})
```

Note: The `analyzer.ConcurrencyOptions` struct field names must match what leakspok defines. Run `grep -A 20 "ConcurrencyOptions" vendor/go.ifoodcorp.com.br/leakspok/analyzer/` to confirm exact field names before implementing. Adjust the field mapping accordingly.

- [ ] **Step 4: Run test again**

Run: `go test ./pkg/anonymizer/ -run TestNewFromConfig_ConcurrencyConfigWired -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/anonymizer/app.go pkg/anonymizer/app_test.go
git commit -m "feat: wire concurrency configuration from env into ByteAnalyzer RunnerOptions"
```

---

## Task 7: Port YAML Rules Loader from V2

**Files:**
- Create: `pkg/config/yaml_loader.go`
- Create: `pkg/config/yaml_loader_test.go`
- Create: `pkg/config/testdata/email_service.yaml`

- [ ] **Step 1: Create test fixture**

Create `pkg/config/testdata/email_service.yaml`:

```yaml
entities:
  - name: EMAIL
    redaction:
      replacement: "<EMAIL_REDACTED>"
  - name: CPF_NUMBER
    redaction:
      replacement: "<CPF_REDACTED>"
    exceptions:
      - reason: "test CPF"
        match:
          operator: equal
          pattern: "000.000.000-00"
```

- [ ] **Step 2: Write failing test**

Create `pkg/config/yaml_loader_test.go`:

```go
package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"anonymizer-service-v2/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRulesLoader_Load_Success(t *testing.T) {
	basePath := filepath.Join("testdata")
	loader := config.NewYAMLRulesLoader(basePath)

	rules, err := loader.Load(context.Background(), "email_service")
	require.NoError(t, err)
	assert.Len(t, rules, 2) // EMAIL + CPF_NUMBER
}

func TestYAMLRulesLoader_Load_FileNotFound(t *testing.T) {
	basePath := filepath.Join("testdata")
	loader := config.NewYAMLRulesLoader(basePath)

	_, err := loader.Load(context.Background(), "nonexistent_service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

func TestYAMLRulesLoader_Load_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "broken_service.yaml"), []byte("not: valid: yaml: ["), 0644)
	require.NoError(t, err)

	loader := config.NewYAMLRulesLoader(dir)
	_, err = loader.Load(context.Background(), "broken_service")
	assert.Error(t, err)
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./pkg/config/ -run TestYAMLRulesLoader -v`
Expected: FAIL — `NewYAMLRulesLoader` undefined

- [ ] **Step 4: Implement YAML rules loader**

Create `pkg/config/yaml_loader.go`:

```go
package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"anonymizer-service-v2/pkg/privacy"

	"go.ifoodcorp.com.br/leakspok/analyzer"
	"gopkg.in/yaml.v3"
)

// YAMLRulesLoader loads privacy rules from YAML files on disk.
// Each service has a file named <service_name>.yaml in the base path.
type YAMLRulesLoader struct {
	basePath string
}

// NewYAMLRulesLoader creates a loader that reads YAML files from basePath.
func NewYAMLRulesLoader(basePath string) *YAMLRulesLoader {
	return &YAMLRulesLoader{basePath: basePath}
}

// Load reads the YAML file for the given service and builds analyzer rules.
func (l *YAMLRulesLoader) Load(_ context.Context, serviceName string) ([]analyzer.Rule, error) {
	filePath := filepath.Join(l.basePath, serviceName+".yaml")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", filePath, err)
	}

	var settings privacy.PrivacySettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse YAML rules for %s: %w", serviceName, err)
	}

	rules, err := privacy.NewRuleBuilder(settings).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build rules for %s: %w", serviceName, err)
	}

	return rules, nil
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/config/ -run TestYAMLRulesLoader -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/config/yaml_loader.go pkg/config/yaml_loader_test.go pkg/config/testdata/
git commit -m "feat: add YAML-based rules loader for standalone operation"
```

---

## Task 8: Content Negotiation — Header Parsing

**Files:**
- Create: `internal/handler/headers.go`
- Create: `internal/handler/headers_test.go`

- [ ] **Step 1: Write tests for header parsing**

Create `internal/handler/headers_test.go`:

```go
package handler

import (
	"testing"

	"anonymizer-service-v2/pkg/privacy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAnonymizeHeaders_ValidEntities(t *testing.T) {
	headers := map[string]string{
		HeaderEntities: "EMAIL,CPF_NUMBER",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Len(t, settings.Entities, 2)
	assert.Equal(t, "EMAIL", settings.Entities[0].Name)
	assert.Equal(t, "CPF_NUMBER", settings.Entities[1].Name)
	// Default strategy: redact with <REDACTED>
	assert.NotNil(t, settings.Entities[0].Redaction)
	assert.Equal(t, "<REDACTED>", settings.Entities[0].Redaction.Replacement)
}

func TestParseAnonymizeHeaders_CustomPlaceholder(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:    "EMAIL",
		HeaderPlaceholder: "[REMOVED]",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Equal(t, "[REMOVED]", settings.Entities[0].Redaction.Replacement)
}

func TestParseAnonymizeHeaders_MaskStrategy(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:   "EMAIL",
		HeaderStrategy:   "mask",
		HeaderMaskChar:   "#",
		HeaderMaskLength: "6",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Nil(t, settings.Entities[0].Redaction)
	assert.NotNil(t, settings.Entities[0].Mask)
	assert.Equal(t, "#", settings.Entities[0].Mask.Replacement)
	assert.Equal(t, 6, settings.Entities[0].Mask.MaxLength)
}

func TestParseAnonymizeHeaders_MissingEntities(t *testing.T) {
	headers := map[string]string{}

	_, err := parseAnonymizeHeaders(headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Anonymize-Entities")
}

func TestParseAnonymizeHeaders_InvalidMaskLength(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:   "EMAIL",
		HeaderStrategy:   "mask",
		HeaderMaskLength: "abc",
	}

	_, err := parseAnonymizeHeaders(headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mask-length")
}

func TestParseAnonymizeHeaders_TrimWhitespace(t *testing.T) {
	headers := map[string]string{
		HeaderEntities: " EMAIL , CPF_NUMBER ",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Equal(t, "EMAIL", settings.Entities[0].Name)
	assert.Equal(t, "CPF_NUMBER", settings.Entities[1].Name)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/ -run TestParseAnonymizeHeaders -v`
Expected: FAIL — undefined constants and function

- [ ] **Step 3: Implement header parsing**

Create `internal/handler/headers.go`:

```go
package handler

import (
	"fmt"
	"strconv"
	"strings"

	"anonymizer-service-v2/pkg/privacy"
)

// Header constants for content negotiation.
const (
	HeaderEntities    = "X-Anonymize-Entities"
	HeaderStrategy    = "X-Anonymize-Strategy"
	HeaderPlaceholder = "X-Anonymize-Placeholder"
	HeaderMaskChar    = "X-Anonymize-Mask-Char"
	HeaderMaskLength  = "X-Anonymize-Mask-Length"

	HeaderDetectedEntities   = "X-Anonymize-Detected-Entities"
	HeaderAnonymizedEntities = "X-Anonymize-Anonymized-Entities"
)

// parseAnonymizeHeaders builds PrivacySettings from X-Anonymize-* request headers.
// headers is a map of header-name → value (caller extracts from http.Request).
func parseAnonymizeHeaders(headers map[string]string) (privacy.PrivacySettings, error) {
	entitiesRaw, ok := headers[HeaderEntities]
	if !ok || strings.TrimSpace(entitiesRaw) == "" {
		return privacy.PrivacySettings{}, fmt.Errorf("%s header is required when no rules are pre-configured", HeaderEntities)
	}

	strategy := strings.ToLower(strings.TrimSpace(headers[HeaderStrategy]))
	if strategy == "" {
		strategy = "redact"
	}

	entityNames := strings.Split(entitiesRaw, ",")
	entities := make([]privacy.EntitySettings, 0, len(entityNames))

	for _, name := range entityNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		entity := privacy.EntitySettings{Name: strings.ToUpper(name)}

		switch strategy {
		case "redact":
			placeholder := headers[HeaderPlaceholder]
			if placeholder == "" {
				placeholder = "<REDACTED>"
			}
			entity.Redaction = &privacy.RedactionSettings{Replacement: placeholder}

		case "mask":
			maskChar := headers[HeaderMaskChar]
			if maskChar == "" {
				maskChar = "*"
			}
			maskLength := 4
			if ml, exists := headers[HeaderMaskLength]; exists && ml != "" {
				parsed, err := strconv.Atoi(ml)
				if err != nil {
					return privacy.PrivacySettings{}, fmt.Errorf("invalid %s value %q: must be an integer", HeaderMaskLength, ml)
				}
				maskLength = parsed
			}
			entity.Mask = &privacy.MaskSettings{Replacement: maskChar, MaxLength: maskLength}

		default:
			return privacy.PrivacySettings{}, fmt.Errorf("invalid %s value %q: must be 'redact' or 'mask'", HeaderStrategy, strategy)
		}

		entities = append(entities, entity)
	}

	return privacy.PrivacySettings{Entities: entities}, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/handler/ -run TestParseAnonymizeHeaders -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add internal/handler/headers.go internal/handler/headers_test.go
git commit -m "feat: add X-Anonymize-* header parsing for text/plain content negotiation"
```

---

## Task 9: Content Negotiation — Text/Plain Codec and Dispatch

**Files:**
- Modify: `internal/handler/handler.go` (add content type dispatch)
- Create: `internal/handler/handler_text_test.go`

- [ ] **Step 1: Write tests for text/plain anonymization**

Create `internal/handler/handler_text_test.go`:

```go
package handler_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"anonymizer-service-v2/internal/handler"
	"anonymizer-service-v2/pkg/privacy"
	privacymock "anonymizer-service-v2/pkg/privacy/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler_Anonymize_TextPlain_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
	h := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("Contact us at john@example.com"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Anonymize-Entities", "EMAIL")
	req.Header.Set("X-Anonymize-Placeholder", "<EMAIL_REDACTED>")
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "<EMAIL_REDACTED>")
	assert.NotContains(t, rec.Body.String(), "john@example.com")
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Detected-Entities"))
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Anonymized-Entities"))
}

func TestHandler_Anonymize_TextPlain_MissingEntities_NoContextRules(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
	h := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("some text"))
	req.Header.Set("Content-Type", "text/plain")
	// No X-Anonymize-Entities header
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-Anonymize-Entities")
}

func TestHandler_Anonymize_UnsupportedContentType(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
	h := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("<xml/>"))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported content type")
}

func TestHandler_Anonymize_NoContentType_DefaultsJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
	h := handler.NewHandler(logger, privacyService, maxBatchSize)

	body := `{"text":"user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
	// No Content-Type header — should default to JSON
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "REDACTED")
}

func TestHandler_AnonymizeBatch_NonJSON_Returns415(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLoader := privacymock.NewMockPrivacyRulesLoader(ctrl)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, mockLoader, logger)
	h := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Contains(t, rec.Body.String(), "batch endpoint only accepts application/json")
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/handler/ -run "TestHandler_Anonymize_TextPlain|TestHandler_Anonymize_Unsupported|TestHandler_Anonymize_NoContentType|TestHandler_AnonymizeBatch_NonJSON" -v`
Expected: FAIL — text/plain path doesn't exist, 415 not returned

- [ ] **Step 3: Implement content negotiation dispatch in handler.go**

Modify the `Anonymize` method in `internal/handler/handler.go` to dispatch on `Content-Type`:

```go
func (h *Handler) Anonymize(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	// Normalize: strip parameters (e.g., "text/plain; charset=utf-8" → "text/plain")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	contentType = strings.ToLower(contentType)

	switch contentType {
	case "text/plain":
		h.anonymizeTextPlain(w, r)
	case "application/json", "":
		h.anonymizeJSON(w, r)
	default:
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			fmt.Sprintf("unsupported content type: %s", contentType))
	}
}
```

Rename the current `Anonymize` body to `anonymizeJSON` (private method). Add `anonymizeTextPlain`:

```go
func (h *Handler) anonymizeTextPlain(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	span, ctx := monitoring.StartSpan(ctx, "anonymize_text_plain")
	defer span.Finish()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Try to build settings from headers
	headerMap := map[string]string{
		HeaderEntities:    r.Header.Get(HeaderEntities),
		HeaderStrategy:    r.Header.Get(HeaderStrategy),
		HeaderPlaceholder: r.Header.Get(HeaderPlaceholder),
		HeaderMaskChar:    r.Header.Get(HeaderMaskChar),
		HeaderMaskLength:  r.Header.Get(HeaderMaskLength),
	}

	var rules []analyzer.Rule

	// Check if headers provide inline settings
	if headerMap[HeaderEntities] != "" {
		settings, err := parseAnonymizeHeaders(headerMap)
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_HEADERS", err.Error())
			return
		}
		if err := config.ValidatePrivacyConfig(settings); err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}
		rules, err = privacy.NewRuleBuilder(settings).Build()
		if err != nil {
			respondError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
			return
		}
	} else {
		// Fallback: check context for pre-injected rules (from plugin middleware)
		ctxRules, ok := anonymizer.RulesFromContext(ctx)
		if !ok || len(ctxRules) == 0 {
			respondError(w, http.StatusBadRequest, "NO_RULES",
				"X-Anonymize-Entities header is required when no rules are pre-configured")
			return
		}
		rules = ctxRules
	}

	responseBuffer := h.bufferPool.GetResponseBuffer()
	defer h.bufferPool.PutResponseBuffer(responseBuffer)

	output, err := h.privacyService.AnonymizeWithRules(ctx, rules, body, responseBuffer)
	if err != nil {
		monitoring.SetError(span, err)
		respondError(w, http.StatusInternalServerError, "ANONYMIZATION_FAILED", err.Error())
		return
	}

	// Build response headers
	detectedEntities := entitySliceToStrings(output.Details.DetectedEntities)
	anonymizedEntities := entitySliceToStrings(output.Details.AnonymizedEntities)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set(HeaderDetectedEntities, strings.Join(detectedEntities, ","))
	w.Header().Set(HeaderAnonymizedEntities, strings.Join(anonymizedEntities, ","))
	w.WriteHeader(http.StatusOK)
	w.Write(responseBuffer.Bytes())
}
```

Add helper (in handler.go or a separate small file):
```go
func entitySliceToStrings(entities []pattern.Entity) []string {
	result := make([]string, 0, len(entities))
	for _, e := range entities {
		result = append(result, string(e))
	}
	sort.Strings(result)
	return result
}
```

Also add 415 check at the top of `AnonymizeBatch`:
```go
func (h *Handler) AnonymizeBatch(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	if contentType != "" && strings.ToLower(contentType) != "application/json" {
		respondError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE",
			"batch endpoint only accepts application/json")
		return
	}
	// ... rest of existing batch logic
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/handler/ -run "TestHandler_Anonymize_TextPlain|TestHandler_Anonymize_Unsupported|TestHandler_Anonymize_NoContentType|TestHandler_AnonymizeBatch_NonJSON" -v`
Expected: All PASS

Note: `TestHandler_Anonymize_TextPlain_Success` depends on `anonymizer.RulesFromContext` which doesn't exist yet. For now, this test uses headers to provide rules inline, so it should pass. The context rules test will be added in Task 10.

- [ ] **Step 5: Run full test suite to check for regressions**

Run: `go test ./...`
Expected: All pass (JSON path unchanged, only dispatch added)

- [ ] **Step 6: Commit**

```bash
git add internal/handler/handler.go internal/handler/handler_text_test.go
git commit -m "feat: add text/plain content negotiation on /api/v1/anonymize"
```

---

## Task 10: Batch Endpoint 415 and Response Header Tests

**Files:**
- Already added in Task 8, verify everything works end-to-end

- [ ] **Step 1: Run the full test suite including integration tests**

Run: `go test ./... -count=1`
Expected: All pass.

- [ ] **Step 2: Commit if any additional fixes were needed**

```bash
git add -A
git commit -m "fix: ensure batch 415 and content negotiation edge cases pass"
```

---

## Task 11: Plugin System — MiddlewareRegistrar and Context Helpers

**Files:**
- Create: `pkg/anonymizer/context.go`
- Create: `pkg/anonymizer/context_test.go`
- Modify: `pkg/anonymizer/plugin.go` (replace interfaces)
- Modify: `pkg/anonymizer/options.go` (remove WithRulesLoader)
- Modify: `pkg/anonymizer/app.go` (wire middleware, remove composite loader)
- Delete: `pkg/anonymizer/composite_loader.go`
- Delete: `pkg/anonymizer/composite_loader_test.go`

- [ ] **Step 1: Write tests for context helpers**

Create `pkg/anonymizer/context_test.go`:

```go
package anonymizer_test

import (
	"context"
	"testing"

	"anonymizer-service-v2/pkg/anonymizer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.ifoodcorp.com.br/leakspok/analyzer"
	"go.ifoodcorp.com.br/leakspok/pattern"
)

func TestWithRules_RoundTrip(t *testing.T) {
	rules := []analyzer.Rule{
		{Matcher: pattern.NewRegexMatcher(pattern.EntityEmail, `test@example\.com`)},
	}

	ctx := anonymizer.WithRules(context.Background(), rules)
	got, ok := anonymizer.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Len(t, got, 1)
}

func TestRulesFromContext_EmptyContext(t *testing.T) {
	_, ok := anonymizer.RulesFromContext(context.Background())
	assert.False(t, ok)
}

func TestWithRules_NilRules(t *testing.T) {
	ctx := anonymizer.WithRules(context.Background(), nil)
	got, ok := anonymizer.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Nil(t, got)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/anonymizer/ -run "TestWithRules|TestRulesFromContext" -v`
Expected: FAIL — `WithRules` and `RulesFromContext` undefined

- [ ] **Step 3: Implement context helpers**

Create `pkg/anonymizer/context.go`:

```go
package anonymizer

import (
	"context"

	"go.ifoodcorp.com.br/leakspok/analyzer"
)

type contextKey struct{}

// WithRules stores pre-built anonymization rules in the context.
// Plugins call this in their middleware to inject rules for the handler.
func WithRules(ctx context.Context, rules []analyzer.Rule) context.Context {
	return context.WithValue(ctx, contextKey{}, rules)
}

// RulesFromContext retrieves rules from the context.
// Returns the rules and true if present, nil and false otherwise.
func RulesFromContext(ctx context.Context) ([]analyzer.Rule, bool) {
	rules, ok := ctx.Value(contextKey{}).([]analyzer.Rule)
	return rules, ok
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/anonymizer/ -run "TestWithRules|TestRulesFromContext" -v`
Expected: PASS

- [ ] **Step 5: Replace plugin interfaces**

Rewrite `pkg/anonymizer/plugin.go`:

```go
package anonymizer

import (
	"go.ifoodcorp.com.br/leakspok/analyzer"
	"log/slog"
	"net/http"
)

// CoreServices provides shared dependencies to plugins.
type CoreServices struct {
	Logger       *slog.Logger
	ByteAnalyzer analyzer.ByteAnalyzer
}

// MiddlewareRegistrar is implemented by plugins that inject middleware
// into the HTTP request chain. The middleware runs before the core handler.
type MiddlewareRegistrar interface {
	Middleware(services CoreServices) func(http.Handler) http.Handler
}
```

- [ ] **Step 6: Update options.go — remove WithRulesLoader**

Rewrite `pkg/anonymizer/options.go`:

```go
package anonymizer

import (
	"github.com/uber-go/tally/v4"
	"go.ifoodcorp.com.br/leakspok/analyzer"
	"log/slog"
)

// Option configures the Service during construction.
type Option func(*Service)

// WithPlugin registers a plugin. The builder detects MiddlewareRegistrar
// via type assertion and wires it accordingly.
func WithPlugin(p any) Option {
	return func(a *Service) {
		a.plugins = append(a.plugins, p)
	}
}

// WithLogger sets a custom logger. If not called, slog.Default() is used.
func WithLogger(l *slog.Logger) Option {
	return func(a *Service) {
		a.logger = l
	}
}

// WithByteAnalyzer sets a custom ByteAnalyzer. If not called, one is
// created internally from environment configuration.
func WithByteAnalyzer(ba analyzer.ByteAnalyzer) Option {
	return func(a *Service) {
		a.byteAnalyzer = &ba
	}
}

// WithMetricsScope sets a tally.Scope for metrics collection.
// If not called, metrics are disabled (NoopScope).
func WithMetricsScope(scope tally.Scope) Option {
	return func(a *Service) {
		a.metricsScope = scope
	}
}
```

- [ ] **Step 7: Delete composite loader**

```bash
rm pkg/anonymizer/composite_loader.go pkg/anonymizer/composite_loader_test.go
```

- [ ] **Step 8: Update app.go — remove composite loader logic, add middleware wiring**

In `pkg/anonymizer/app.go`:
- Remove `rulesLoader` field from `Service` struct
- Remove all `RulesLoaderProvider` logic from `NewFromConfig`
- Remove `compositeRulesLoader` usage
- Pass `nil` for `rulesLoader` to `privacy.NewService` (rules come from context or inline settings now)
- In `Handler()`: wire plugin middleware before routes

Update the `Handler()` method:
```go
func (a *Service) Handler() http.Handler {
	const healthPath = "/health"
	router := chi.NewRouter()

	// Standard middleware
	router.Use(middleware.RealIP)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(middleware.RequestIDHeader)
			if requestID == "" {
				requestID = uuid.NewString()
			}
			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	})
	router.Use(middleware.Recoverer)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := leakspokmonitoring.WithPatternMonitoring(r.Context(), a.envConfig.PatternMonitoringEnabled)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	router.Use(handler.AccessLog(a.logger, []string{healthPath}))

	// Plugin middleware (runs before core handler)
	coreServices := CoreServices{
		Logger:       a.logger,
		ByteAnalyzer: *a.byteAnalyzer,
	}
	for _, p := range a.plugins {
		if mr, ok := p.(MiddlewareRegistrar); ok {
			router.Use(mr.Middleware(coreServices))
		}
	}

	// Routes
	router.Get(healthPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	})

	var h *handler.Handler
	if a.metricsScope != nil && a.metricsScope != tally.NoopScope {
		metrics := handler.NewPrivacyMetrics(a.metricsScope)
		h = handler.NewHandlerWithMetrics(a.logger, a.privacyService, a.envConfig.MaxBatchSize, metrics)
	} else {
		h = handler.NewHandler(a.logger, a.privacyService, a.envConfig.MaxBatchSize)
	}

	router.Route("/api/v1", func(r chi.Router) {
		r.Post("/anonymize", h.Anonymize)
		r.Post("/anonymize/batch", h.AnonymizeBatch)
	})

	return router
}
```

- [ ] **Step 9: Update app_test.go — replace old plugin tests with MiddlewareRegistrar tests**

Replace `mockRouteRegistrar` and `mockRulesLoaderProvider` tests with:

```go
// mockMiddlewarePlugin is a test plugin that injects a header into the response.
type mockMiddlewarePlugin struct {
	called bool
}

func (m *mockMiddlewarePlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
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

	app, err := anonymizer.NewFromConfig(context.Background(),
		anonymizer.WithPlugin(plugin),
	)
	require.NoError(t, err)

	h := app.Handler()
	assert.True(t, plugin.called)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "true", rec.Header().Get("X-Plugin-Active"))
}

func TestHandler_PluginInjectsContextRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Plugin that injects EMAIL rules into context
	plugin := &contextRulesPlugin{}

	app, err := anonymizer.NewFromConfig(context.Background(),
		anonymizer.WithPlugin(plugin),
	)
	require.NoError(t, err)

	h := app.Handler()

	// text/plain request with NO X-Anonymize-Entities — relies on plugin context rules
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("Contact john@example.com"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), "john@example.com")
}
```

Define `contextRulesPlugin` in the same test file:

```go
// contextRulesPlugin injects EMAIL redaction rules into every request context.
type contextRulesPlugin struct{}

func (p *contextRulesPlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
	// Build EMAIL redaction rules once at middleware creation time
	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<PLUGIN_REDACTED>"}},
		},
	}
	rules, _ := privacy.NewRuleBuilder(settings).Build()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := anonymizer.WithRules(r.Context(), rules)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

- [ ] **Step 10: Run all tests**

Run: `go test ./...`
Expected: All pass.

- [ ] **Step 11: Commit**

```bash
git add -A
git commit -m "feat: replace RouteRegistrar and RulesLoaderProvider with single MiddlewareRegistrar interface"
```

---

## Task 12: Example Plugin

**Files:**
- Create: `examples/plugin/main.go`
- Create: `examples/plugin/header_plugin.go`

- [ ] **Step 1: Create the example plugin implementation**

Create `examples/plugin/header_plugin.go`:

```go
package main

import (
	"log/slog"
	"net/http"

	"anonymizer-service-v2/pkg/anonymizer"
	"anonymizer-service-v2/pkg/privacy"

	"go.ifoodcorp.com.br/leakspok/analyzer"
)

// ServiceRulesPlugin demonstrates a plugin that reads a service name header,
// looks up rules for that service, and injects them into the request context.
type ServiceRulesPlugin struct {
	// rulesByService simulates a rule config store (in practice, this would be an API client).
	rulesByService map[string]privacy.PrivacySettings
}

func NewServiceRulesPlugin() *ServiceRulesPlugin {
	return &ServiceRulesPlugin{
		rulesByService: map[string]privacy.PrivacySettings{
			"email-service": {
				Entities: []privacy.EntitySettings{
					{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<EMAIL>"}},
					{Name: "CPF_NUMBER", Redaction: &privacy.RedactionSettings{Replacement: "<CPF>"}},
				},
			},
			"payment-service": {
				Entities: []privacy.EntitySettings{
					{Name: "CREDIT_CARD", Redaction: &privacy.RedactionSettings{Replacement: "<CARD>"}},
				},
			},
		},
	}
}

func (p *ServiceRulesPlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceName := r.Header.Get("X-Service-Name")
			if serviceName == "" {
				next.ServeHTTP(w, r)
				return
			}

			settings, ok := p.rulesByService[serviceName]
			if !ok {
				services.Logger.Warn("unknown service", slog.String("service", serviceName))
				next.ServeHTTP(w, r)
				return
			}

			rules, err := privacy.NewRuleBuilder(settings).Build()
			if err != nil {
				services.Logger.Error("failed to build rules", slog.String("error", err.Error()))
				next.ServeHTTP(w, r)
				return
			}

			ctx := anonymizer.WithRules(r.Context(), rules)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Verify interface compliance at compile time.
var _ anonymizer.MiddlewareRegistrar = (*ServiceRulesPlugin)(nil)
```

- [ ] **Step 2: Create the example main.go**

Create `examples/plugin/main.go`:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"anonymizer-service-v2/pkg/anonymizer"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	plugin := NewServiceRulesPlugin()

	app, err := anonymizer.NewFromConfig(ctx,
		anonymizer.WithLogger(logger),
		anonymizer.WithPlugin(plugin),
	)
	if err != nil {
		logger.Error("failed to create service", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("starting anonymizer with service-rules plugin")
	if err := app.ListenAndServe(ctx); err != nil {
		logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
```

- [ ] **Step 3: Verify example compiles**

Run: `go build ./examples/plugin/`
Expected: Success

- [ ] **Step 4: Commit**

```bash
git add examples/plugin/
git commit -m "docs: add example plugin demonstrating MiddlewareRegistrar with context rules"
```

---

## Task 13: Plugin Documentation

**Files:**
- Create: `docs/plugins.md`

- [ ] **Step 1: Write plugin developer guide**

Create `docs/plugins.md` covering:
1. The `MiddlewareRegistrar` interface
2. `CoreServices` struct
3. Context helpers (`WithRules`, `RulesFromContext`)
4. Step-by-step: writing a middleware plugin
5. Rule precedence (inline > context)
6. Testing plugins with httptest
7. Running the example

- [ ] **Step 2: Commit**

```bash
git add docs/plugins.md
git commit -m "docs: add plugin developer guide"
```

---

## Task 14: Final Cleanup and Full Validation

**Files:**
- Modify: `pkg/anonymizer/app_test.go` (rename GenPlat test)
- Verify: all files compile and test green

- [ ] **Step 1: Rename GenPlat reference in test**

In `pkg/anonymizer/app_test.go`, rename `TestHandler_GenplatRouteReturns404` to `TestHandler_UnknownRouteReturns404` and remove any GenPlat-specific comments.

- [ ] **Step 2: Run linter**

Run: `golangci-lint run ./...` (if available) or `go vet ./...`
Expected: No errors

- [ ] **Step 3: Run full test suite**

Run: `go test -race ./...`
Expected: All pass with no race conditions

- [ ] **Step 4: Verify no foodsec-go-sdk imports remain**

Run: `grep -r "foodsec" --include="*.go" . | grep -v vendor | grep -v "_test.go"`
Expected: Zero matches in production code. (Test files may still reference mocks — check and fix if found.)

- [ ] **Step 5: Verify no DataDog direct imports remain**

Run: `grep -r "DataDog\|dd-trace" --include="*.go" . | grep -v vendor`
Expected: Zero matches.

- [ ] **Step 6: Verify example plugin builds**

Run: `go build ./examples/plugin/`
Expected: Success

- [ ] **Step 7: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup — rename GenPlat test, verify no internal deps"
```

---

## Notes on V2 Feature Parity

- **anonymizeFullRead optimization**: The text/plain path uses `io.ReadAll(r.Body)` which IS the full-read strategy. The JSON path uses `json.NewDecoder(r.Body).Decode` which also reads the full body. No separate task needed — this optimization is inherent in the implementation.
- **Entity-filtered handler**: Entity filtering is already implemented in `privacy/service.go:filterRules()`. The content negotiation headers (`X-Anonymize-Entities`) select which entities to build rules for — achieving the same result. The `privacy.NewRuleBuilder(settings).Build()` only creates rules for the entities listed in settings.
- **YAML Rules Loader**: Task 7 implements this as a standalone loader. It can be used as a default rules source via plugin middleware or directly via `WithRulesLoader` option (if re-added for convenience).

---

## Summary of Commits

| # | Message | Phase |
|---|---------|-------|
| 1 | `refactor: delete stale duplicate packages` | 2A |
| 2 | `refactor: replace foodsec-go-sdk error responses with stdlib JSON helper` | 2A |
| 3 | `refactor: replace DataDog tracing and foodsec access log with OTel and slog` | 2A |
| 4 | `refactor: remove foodsec-go-sdk dependency, use stdlib slog and signal handling` | 2A |
| 5 | `build: remove foodsec-go-sdk and DataDog deps, add OpenTelemetry` | 2A |
| 6 | `feat: wire concurrency configuration from env into ByteAnalyzer RunnerOptions` | 2A |
| 7 | `feat: add YAML-based rules loader for standalone operation` | 2A |
| 8 | `feat: add X-Anonymize-* header parsing for text/plain content negotiation` | 2B |
| 9 | `feat: add text/plain content negotiation on /api/v1/anonymize` | 2B |
| 10 | `fix: ensure batch 415 and content negotiation edge cases pass` | 2B |
| 11 | `feat: replace RouteRegistrar and RulesLoaderProvider with single MiddlewareRegistrar interface` | 2C |
| 12 | `docs: add example plugin demonstrating MiddlewareRegistrar with context rules` | 2C |
| 13 | `docs: add plugin developer guide` | 2C |
| 14 | `chore: final cleanup — rename GenPlat test, verify no internal deps` | 2C |
