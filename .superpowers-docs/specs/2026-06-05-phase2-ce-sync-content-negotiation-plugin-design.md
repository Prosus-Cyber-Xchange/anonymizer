# Phase 2 Design: CE Sync, Content Negotiation & Plugin System

**Date:** 2026-06-05
**Scope:** `anonymizer-service-v2-ce/` — Phase 2 of the Open-Source Release Plan
**Approach:** Sequential subphases (2A → 2B → 2C)

---

## Decisions Made

| Decision | Choice |
|----------|--------|
| Leakspok dependency | Stays as-is (`go.ifoodcorp.com.br/leakspok`). Module path updated later. |
| foodsec-go-sdk | Remove in this phase. Replace with stdlib equivalents. |
| Package layout | Keep `pkg/*`. Delete `internal/api/`, `internal/privacy/`, `internal/config/`. |
| DataDog | Replace with OpenTelemetry. |
| V2 feature port | Port all four: YAML loader, concurrency config, anonymizeFullRead, entity-filtered handler. |
| Plugin interfaces | Single interface: `MiddlewareRegistrar`. No `RouteRegistrar`, no `RulesLoaderProvider`. |
| Rule precedence | Inline settings (request) take precedence over context-injected rules (middleware). |
| Custom logger | `WithLogger(*slog.Logger)` propagates to plugins via `CoreServices`. |

---

## Phase 2A — Sync & Cleanup

### 2A.1 Delete Stale Duplicate Packages

Remove entirely:
- `internal/api/`
- `internal/privacy/`
- `internal/config/`

After deletion, `internal/` retains:
- `internal/handler/` — HTTP layer (handler, router, pool, metrics)
- `internal/monitoring/` — Observability helpers

All imports update to reference `pkg/config` and `pkg/privacy` exclusively.

### 2A.2 Replace foodsec-go-sdk

| SDK Usage | Replacement |
|-----------|-------------|
| `entrypoint.Do(run)` in `main.go` | `signal.NotifyContext` + `os.Exit` (~20 LOC) |
| `logging.NewSLogger()` / `logging.WithAttribute()` | `slog.New(slog.NewJSONHandler(os.Stdout, opts))` / `slog.With()` |
| `foodsecserver.Options` (Port, Host, GracefulShutdown) | Custom `ServerConfig` struct in `pkg/config/` with same env vars |
| `foodsecserver.AccessLog` middleware | Custom chi middleware using `slog` (~30 LOC) |
| `foodsechttp.RespondError` | Custom `respondError(w, status, msg)` helper (~15 LOC) |
| `foodsecconfig` import in `env.go` | Remove (already using `caarlos0/env/v11`) |

### 2A.3 Replace DataDog with OpenTelemetry

| DataDog Component | OTel Replacement |
|-------------------|------------------|
| `ddtrace/tracer.Start()` in `main.go` | `otel.SetTracerProvider()` with OTLP exporter |
| `dd-trace-go/contrib/chi.v5` middleware | `otelhttp` middleware (or `otelchi` contrib) |
| `monitoring.StartSpan()` / `SetError()` | `otel.Tracer().Start()` / `span.RecordError()` |
| `profiler.Start()` | Remove (pprof endpoint optional) |
| tally metrics | Keep tally for now (vendor-neutral) |

The `internal/monitoring/` package becomes a thin wrapper over OTel APIs.

### 2A.4 Port V2 Features

1. **YAML Rules Loader** — Port `internal/config/yaml.go` from v2 into `pkg/config/yaml_loader.go`. Enables standalone operation by loading rules from YAML files on disk.

2. **Concurrency Configuration** — Add to `pkg/config/env.go` under the `Privacy` struct:

   | Field | Env Var | Type | Default | Description |
   |-------|---------|------|---------|-------------|
   | `ConcurrencyEnabled` | `PRIVACY_CONCURRENCY_ENABLED` | `bool` | `false` | Master toggle for concurrent processing |
   | `ConcurrencyTokenProcessing` | `PRIVACY_CONCURRENCY_TOKEN_PROCESSING` | `bool` | `false` | Enable token-level parallelism |
   | `ConcurrencyRuleProcessing` | `PRIVACY_CONCURRENCY_RULE_PROCESSING` | `bool` | `false` | Enable rule-level parallelism |
   | `ConcurrencyRuleRunnerPoolSize` | `PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE` | `int` | `0` (unlimited) | Goroutine pool size for rule runners |
   | `ConcurrencyTokenPoolSize` | `PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE` | `int` | `0` (unlimited) | Goroutine pool size for token processing |
   | `ConcurrencyMaxGoroutineIdleTimeout` | `PRIVACY_CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT` | `time.Duration` | `10s` | Idle timeout before reclaiming pool goroutines |

   Wire these into `RunnerOptions` in `NewFromConfig()`.

3. **`anonymizeFullRead` optimization** — Port the full-body-read strategy (payloads < 64KB) into `internal/handler/handler.go`. Uses buffer pool, reduces allocations.

4. **Entity-filtered handler** — Port as a header-based filtering path. Reads entity list from `X-Anonymize-Entities` header, filters rules accordingly. Integrated into content negotiation (Phase 2B).

### 2A.5 Update go.mod

- Remove `code.ifoodcorp.com.br/ifood/security/libs/go/foodsec-go-sdk`
- Remove `github.com/DataDog/dd-trace-go/v2`
- Add `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`
- Keep `go.ifoodcorp.com.br/leakspok` as-is (deferred)

---

## Phase 2B — Content Negotiation

### Architecture

Content negotiation is a dispatcher layer in the handler. The handler inspects `Content-Type` and delegates to the appropriate codec (JSON or plain text).

```
Request → Router → Handler.Anonymize()
                       ├─ Content-Type: application/json → decodeJSON() → anonymize → encodeJSON()
                       └─ Content-Type: text/plain       → decodeText() → anonymize → encodeText()
```

### New Files

| File | Purpose |
|------|---------|
| `internal/handler/codec.go` | `Codec` interface + JSON/PlainText implementations |
| `internal/handler/codec_json.go` | JSON decode/encode (extracted from current handler) |
| `internal/handler/codec_text.go` | Plain text decode (body = text) + header parsing for settings |
| `internal/handler/headers.go` | Constants and parsing for `X-Anonymize-*` headers |

### Request Flow — text/plain

1. Client sends `POST /api/v1/anonymize` with:
   - `Content-Type: text/plain`
   - `X-Anonymize-Entities: EMAIL,CPF_NUMBER` (required unless context rules are injected by middleware)
   - `X-Anonymize-Strategy: redact` (optional, default: `redact`)
   - `X-Anonymize-Placeholder: <REDACTED>` (optional)
   - Body: raw text to anonymize

2. Handler detects `text/plain`, delegates to text codec:
   - Reads raw body as the input text
   - Parses `X-Anonymize-*` headers into `PrivacySettings`
   - Validates settings (same `ValidatePrivacyConfig()` path)
   - Builds rules, anonymizes

3. Response:
   - `Content-Type: text/plain; charset=utf-8`
   - `X-Anonymize-Detected-Entities: EMAIL,CPF_NUMBER`
   - `X-Anonymize-Anonymized-Entities: EMAIL`
   - Body: anonymized plain text

### Header Specification

**Request headers (text/plain mode):**

| Header | Required | Default | Values |
|--------|----------|---------|--------|
| `X-Anonymize-Entities` | Yes (unless context rules exist) | — | Comma-separated: `EMAIL,CPF_NUMBER,CREDIT_CARD,...` |
| `X-Anonymize-Strategy` | No | `redact` | `redact` or `mask` |
| `X-Anonymize-Placeholder` | No | `<REDACTED>` | Any string (redact mode) |
| `X-Anonymize-Mask-Char` | No | `*` | Single character (mask mode) |
| `X-Anonymize-Mask-Length` | No | `4` | Integer > 0 (mask mode) |

**Response headers:**

| Header | Always Present | Description |
|--------|---------------|-------------|
| `X-Anonymize-Detected-Entities` | Yes | Comma-separated detected entity types |
| `X-Anonymize-Anonymized-Entities` | Yes | Comma-separated anonymized entity types |

### Rule Resolution Precedence

```
POST /api/v1/anonymize
  ├─ Has inline settings? (JSON body.settings OR X-Anonymize-Entities header)
  │     YES → Build rules from inline settings. Highest priority.
  │     NO  → Read rules from context (injected by plugin middleware)
  │              └─ Found? → Use context rules
  │              └─ Not found? → 400 Bad Request ("no anonymization rules provided")
  └─ Anonymize with resolved rules
```

### Error Cases

| Condition | Status | Body |
|-----------|--------|------|
| No inline settings AND no context rules | `400 Bad Request` | `{"error": "no anonymization rules provided"}` |
| text/plain with no `X-Anonymize-Entities` AND no context rules | `400 Bad Request` | `{"error": "X-Anonymize-Entities header is required when no rules are pre-configured"}` |
| Invalid entity name in header | `400 Bad Request` | `{"error": "unknown entity: FOO"}` |
| Unsupported Content-Type on `/api/v1/anonymize` | `415 Unsupported Media Type` | `{"error": "unsupported content type: application/xml"}` |
| Non-JSON Content-Type on `/api/v1/anonymize/batch` | `415 Unsupported Media Type` | `{"error": "batch endpoint only accepts application/json"}` |
| No Content-Type header | Defaults to `application/json` | — |

### Batch Endpoint

`/api/v1/anonymize/batch` remains JSON-only. Any non-JSON content type returns `415`. No changes to batch logic.

---

## Phase 2C — Plugin System Hardening

### Single Plugin Interface

```go
// MiddlewareRegistrar — plugins inject middleware into the request chain.
type MiddlewareRegistrar interface {
    Middleware(services CoreServices) func(http.Handler) http.Handler
}
```

No `RouteRegistrar`. No `RulesLoaderProvider`. One interface.

### CoreServices

```go
type CoreServices struct {
    Logger       *slog.Logger
    ByteAnalyzer analyzer.ByteAnalyzer
}
```

Minimal surface. Plugins receive the logger and byte analyzer — enough to fetch rules and do their own processing.

### Custom Logger

`WithLogger(*slog.Logger)` propagates to plugins via `CoreServices.Logger`:

```go
app, err := anonymizer.NewFromConfig(ctx,
    anonymizer.WithLogger(customLogger),  // used by service AND passed to plugins
    anonymizer.WithPlugin(myPlugin),
)
```

If no logger is provided, defaults to `slog.Default()`.

### Context Helpers for Rules Injection

The `pkg/anonymizer` package exports context utilities:

```go
// pkg/anonymizer/context.go
func WithRules(ctx context.Context, rules []analyzer.Rule) context.Context
func RulesFromContext(ctx context.Context) ([]analyzer.Rule, bool)
```

Plugins call `WithRules()` in their middleware. The handler calls `RulesFromContext()` as a fallback when no inline settings are present.

### Plugin Registration

```go
anonymizer.NewFromConfig(ctx,
    anonymizer.WithPlugin(myPlugin),  // type-asserts for MiddlewareRegistrar
)

// Internal wiring:
// If plugin implements MiddlewareRegistrar → middleware added to chi stack (before core handler)
// Multiple plugins → middleware applied in registration order
```

### Removals

- Delete `pkg/anonymizer/composite_loader.go` and test
- Delete `RulesLoaderProvider` interface from `pkg/anonymizer/plugin.go`
- Delete `RouteRegistrar` interface from `pkg/anonymizer/plugin.go`
- Remove route-mounting logic from `Service.Handler()`

### Example Plugin

```
examples/plugin/
├── main.go              # Composes CE + example plugin
├── header_plugin.go     # Reads X-Service-Name, fetches/builds rules, injects into ctx
└── README.md
```

The example plugin:
- Middleware reads `X-Service-Name` header
- Builds rules for that service (hardcoded map simulating an API)
- Calls `anonymizer.WithRules(ctx, rules)` to inject into context
- Core handler uses those rules when no inline settings are present

### Plugin Docs (`docs/plugins.md`)

1. The `MiddlewareRegistrar` interface
2. Pattern: header extraction → rule fetching → `WithRules(ctx, rules)`
3. Using `anonymizer.RulesFromContext(ctx)` in custom logic
4. Caching strategies (middleware-level caching for external API calls)
5. Testing plugins with `httptest`

---

## Impact on Goal 3 (GenPlat Plugin)

The GenPlat plugin becomes trivially simple:

```go
type GenplatPlugin struct {
    metadataClient *MetadataAPIClient
}

func (p *GenplatPlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            serviceName := r.Header.Get("X-Ifood-Privacy-Service-Name")
            if serviceName != "" {
                rules, err := p.metadataClient.FetchRules(r.Context(), serviceName)
                if err == nil {
                    r = r.WithContext(anonymizer.WithRules(r.Context(), rules))
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

No dedicated route. No rules loader interface. The internal service uses `/api/v1/anonymize` with `text/plain` content negotiation + the GenPlat middleware injecting rules.

---

## File Changes Summary

### Files to Delete
- `internal/api/` (entire directory)
- `internal/privacy/` (entire directory)
- `internal/config/` (entire directory)
- `pkg/anonymizer/composite_loader.go`
- `pkg/anonymizer/composite_loader_test.go`

### Files to Create
- `internal/handler/codec.go` — Codec interface
- `internal/handler/codec_json.go` — JSON codec
- `internal/handler/codec_text.go` — Plain text codec
- `internal/handler/headers.go` — X-Anonymize-* header constants and parsing
- `pkg/anonymizer/context.go` — WithRules/RulesFromContext helpers
- `pkg/config/yaml_loader.go` — YAML-based rules loader
- `pkg/config/server.go` — ServerConfig (replaces foodsecserver.Options)
- `internal/handler/access_log.go` — slog-based access log middleware (replaces foodsec)
- `internal/handler/errors.go` — respondError helper (replaces foodsechttp)
- `examples/plugin/main.go`
- `examples/plugin/header_plugin.go`
- `examples/plugin/README.md`
- `docs/plugins.md`

### Files to Modify Heavily
- `cmd/server/main.go` — Remove foodsec-go-sdk entrypoint, DataDog; add OTel + signal handling
- `pkg/anonymizer/app.go` — Remove route-mounting logic, remove composite loader wiring, add middleware wiring
- `pkg/anonymizer/plugin.go` — Replace interfaces with single `MiddlewareRegistrar`
- `pkg/anonymizer/options.go` — Clean up options (remove `WithRulesLoader`)
- `pkg/config/env.go` — Add concurrency config, server config, remove DataDog config
- `internal/handler/handler.go` — Add content negotiation dispatch, anonymizeFullRead, rule precedence logic
- `internal/handler/router.go` — Replace DataDog middleware with OTel, remove foodsec access log
- `internal/monitoring/tracer.go` — Replace DataDog with OTel
- `go.mod` — Remove foodsec-go-sdk, DataDog; add OTel

---

## Testing Strategy

### Phase 2A Tests
- All existing tests pass after SDK removal (no behavior change)
- YAML loader: unit tests with fixture files
- Concurrency config: verify RunnerOptions wiring
- anonymizeFullRead: benchmark showing allocation reduction

### Phase 2B Tests
- `text/plain` request → correct anonymization + response headers
- `application/json` request → existing behavior preserved (regression)
- Missing `X-Anonymize-Entities` on text/plain → 400
- Unsupported Content-Type → 415
- No Content-Type → defaults to JSON
- Batch with non-JSON → 415
- Rule precedence: inline settings override context rules

### Phase 2C Tests
- Plugin middleware executes before handler
- `WithRules`/`RulesFromContext` round-trips correctly
- Multiple plugins apply middleware in registration order
- Plugin receives correct CoreServices (logger, analyzer)
- No inline settings + no context rules → 400
- Example plugin builds and runs
