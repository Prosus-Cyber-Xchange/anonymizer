# Phase 4 Design: Comprehensive Documentation

**Date:** 2026-06-06
**Scope:** `anonymizer-service-v2-ce/docs/` — Goal 4 of the Open-Source Release Plan
**Approach:** Plain Markdown files in flat `docs/` directory, no static site generator

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Format | Plain Markdown | User preference; avoids build step and tooling dependency |
| Directory structure | Flat `docs/` (no subdirectories) | ~15 files total; grouping adds noise without navigation value |
| Leakspok API reference (topic #11) | Scoped out | Lives in separate `leakspok-ce` repo; this docs just link to it |
| Deduplication strategy | Entity tables and error tables in ONE file each; endpoint docs and README link to canonical pages | Prevents drift from copy-paste duplication |
| Getting started audience | External user with Docker; no Taskfile dependency | `task` is a contributor tool; first-time users just need Docker or Go |

## Files — 10 New + 3 Updated + 1 New Docker Compose

### New Files

| # | File | Purpose | Rough Size |
|---|------|---------|-------------|
| 1 | `docs/getting-started.md` | Zero to first request | ~80 lines |
| 2 | `docs/entities.md` | Single-source entity reference | ~60 lines |
| 3 | `docs/redaction.md` | Redaction vs mask strategies | ~50 lines |
| 4 | `docs/content-negotiation.md` | JSON vs text/plain, header reference | ~90 lines |
| 5 | `docs/errors.md` | Complete error code reference | ~60 lines |
| 6 | `docs/deployment.md` | Docker, docker-compose, Kubernetes | ~100 lines |
| 7 | `docs/configuration.md` | All env vars organized by domain | ~120 lines |
| 8 | `docs/architecture.md` | Mermaid diagram + component descriptions | ~80 lines |
| 9 | `docs/observability.md` | Logging, metrics, tracing, health check | ~90 lines |
| 10 | `docs/openapi.yaml` | OpenAPI 3.1 specification | ~350 lines |

### Additional New File

| File | Purpose |
|------|---------|
| `docker-compose.yml` (project root) | Spins up Redis + anonymizer service; referenced by `getting-started.md` |

### Updated Files

| File | Changes |
|------|---------|
| `docs/anonymize.md` | Remove inline entity table, error table → link to `entities.md` / `errors.md` |
| `docs/batch.md` | Remove inline entity table, error table → link to `entities.md` / `errors.md` |
| `docs/plugins.md` | Minor: fix stale package paths (`pkg/anonymizer` → `pkg/server`), cross-link to `architecture.md` |
| `README.md` | Trim: remove duplicated entity/error/configuration tables, point to `docs/` instead. Keep intro, quick start snippet, feature list, license. |

## File Content Specifications

### 1. `docs/getting-started.md`

**Purpose:** Take a new user from zero to the first successful anonymization request.

**Sections:**

- **Overview:** The anonymizer is designed for high-performance processing of large text payloads, such as AI prompts. Every design choice — byte-level processing, buffer pooling, optional concurrency, and pluggable caching — targets low latency and minimal resource consumption.
- **Prerequisites:** Docker (recommended) or Go 1.22+
- **Option 1: Docker:** `docker compose up` (requires `docker-compose.yml` in repo root), wait for healthy, done
- **Option 2: Manual (Go):** `cp .env.example .env`, `go run cmd/server/main.go`
- **Verify:** `curl http://localhost:8080/health` → `{"status":"healthy"}`
- **First request — JSON:** Simple `curl` with EMAIL redaction, shows request and response
- **First request — text/plain:** `curl` with `X-Anonymize-Entities: EMAIL` header, shows raw text response and `X-Anonymize-Detected-Entities` response header. The text/plain mode is ideal for piping large text directly without JSON overhead.
- **First batch request:** `curl` with two items, shows request and array response
- **Using as a library:** 3-line Go snippet with import + `NewFromConfig` + `ListenAndServe`
- **Next steps:** Bullet links to `content-negotiation.md`, `entities.md`, `configuration.md`, `plugins.md`

### 2. `docs/entities.md`

**Purpose:** Canonical entity type reference. This is the ONE place entity information lives.

**Sections:**

- **Full entity table:**

  | Name | Aliases | Description | Example Detections |
  |------|---------|-------------|-------------------|
  | `EMAIL` | — | Email addresses | `user@example.com` |
  | `CPF_NUMBER` | — | Brazilian CPF with check-digit validation | `123.456.789-09`, `12345678909` |
  | `CNPJ_NUMBER` | — | Brazilian CNPJ with check-digit validation | `12.345.678/0001-90` |
  | `IP_ADDRESS` | `IP` | IPv4 and IPv6 | `192.168.1.1`, `::1` |
  | `IPV4` | — | IPv4 only | `192.168.1.1` |
  | `IPV6` | — | IPv6 only | `::1`, `2001:db8::1` |
  | `CREDIT_CARD` | — | Credit card numbers | `4111-1111-1111-1111` |
  | `PHONE` | — | Phone numbers, international and Brazilian formats | `+55 11 99999-9999` |
  | `LINK` | `URL` | URLs and hyperlinks | `https://example.com` |
  | `SSN` | — | US Social Security Numbers | `123-45-6789` |
  | `ADDRESS` | — | Street addresses | `123 Main St, Springfield` |
  | `BANK_INFO` | — | Banking info including IBAN | `DE89 3704 0044` |
  | `UUID` | — | UUIDs and GUIDs | `550e8400-e29b-41d4-a716-446655440000` |

- **Case insensitivity note:** `email`, `Email`, `EMAIL` all valid
- **Custom entities:** Brief note that extending entity detection requires changes to the leakspok library (link to leakspok docs)

### 3. `docs/redaction.md`

**Purpose:** Standalone reference for the two anonymization strategies.

**Sections:**

- **Overview:** Two strategies: redact (full replacement) and mask (partial replacement)
- **Redaction:**
  - What it does: replaces entire matched value with a fixed string
  - `replacement` field (required, non-empty string)
  - Before/after example: `john@example.com` → `<EMAIL>`
- **Mask:**
  - What it does: replaces first N characters with a masking character
  - `replacement` field (required, single character typically)
  - `maxLength` field (required, must be > 0)
  - Before/after example with `replacement: "*"`, `maxLength: 4` on CPF `123.456.789-09` → `***.456.789-09`
- **Precedence:** If both `redaction` and `mask` are defined, redaction wins
- **Per-entity configuration:** Each entity can independently choose redaction or mask
- **See also:** Link to `anonymize.md` for request/response format examples

### 4. `docs/content-negotiation.md`

**Purpose:** Complete reference for JSON vs text/plain request modes.

**Sections:**

- **Overview:** When to use each mode; JSON for structured settings, text/plain for simple header-based rules
- **JSON mode (`Content-Type: application/json`):**
  - Request body structure overview
  - Response format: JSON with `anonymized_text`, `detected_entities`, `anonymized_entities`
  - Link to `anonymize.md` for full details
- **text/plain mode (`Content-Type: text/plain`):**
  - Request body: raw text to anonymize
  - Request headers table:

    | Header | Required | Default | Description |
    |--------|----------|---------|-------------|
    | `X-Anonymize-Entities` | Yes | — | Comma-separated entity list: `EMAIL,CPF_NUMBER` |
    | `X-Anonymize-Strategy` | No | `redact` | `redact` or `mask` |
    | `X-Anonymize-Placeholder` | No | `<REDACTED>` | Replacement string for redaction |
    | `X-Anonymize-Mask-Char` | No | `*` | Masking character |
    | `X-Anonymize-Mask-Length` | No | `4` | Characters to preserve in mask mode |

  - Response: raw anonymized text body, entity info in response headers:
    - `X-Anonymize-Detected-Entities`
    - `X-Anonymize-Anonymized-Entities`
  - Example: full `curl` + request/response pair
- **Default behavior:** No `Content-Type` header → treated as JSON
- **Unsupported types:** Return `415 Unsupported Media Type` with `UNSUPPORTED_MEDIA_TYPE` error
- **Batch endpoint:** Only `application/json` accepted; non-JSON returns 415
- **Rule precedence in text/plain mode:**
  1. `X-Anonymize-Entities` header (highest)
  2. Context-injected rules (from plugin middleware)
  3. 400 `NO_RULES` error if neither present

### 5. `docs/errors.md`

**Purpose:** Single source of truth for all error codes. Endpoint docs link here instead of duplicating.

**Sections:**

- **Error response format:** Always JSON: `{"code": "ERROR_CODE", "error": "Human-readable message"}`
- **Error code table:**

  | HTTP Status | Code | Cause | Resolution |
  |-------------|------|-------|------------|
  | `400` | `INVALID_REQUEST` | Malformed JSON body | Check request body is valid JSON |
  | `400` | `INVALID_SETTINGS` | Settings validation failed (empty entities, missing redaction/mask, invalid exception operator, unsupported entity) | Check entity names, ensure each entity has `redaction` or `mask` |
  | `400` | `BATCH_SIZE_EXCEEDED` | Batch items > `MAX_BATCH_SIZE` (default 100) | Reduce batch size or increase `MAX_BATCH_SIZE` |
  | `400` | `NO_RULES` | text/plain request without `X-Anonymize-Entities` and no context rules | Add `X-Anonymize-Entities` header or configure plugin |
  | `415` | `UNSUPPORTED_MEDIA_TYPE` | Unsupported `Content-Type` on `/anonymize` or non-JSON on `/batch` | Use `application/json` or `text/plain` |
  | `500` | `ANONYMIZATION_FAILED` | Unexpected internal error during anonymization | Check server logs for details |

- **See also:** Links to endpoint docs for per-endpoint error context

### 6. `docs/deployment.md`

**Purpose:** Guide for running the service in production environments.

**Sections:**

- **Docker:**
  - Build: `docker build -t anonymizer .`
  - Run: `docker run -p 8080:8080 --env-file .env anonymizer`
  - docker-compose: reference `docker-compose.yml` at repo root (Redis + anonymizer, health checks)
- **Kubernetes:**
  - Minimal Deployment YAML: single replica, env vars from ConfigMap
  - Minimal Service YAML: ClusterIP, port 8080
  - Readiness probe: `GET /health`
  - ConfigMap for env vars
- **Scaling considerations:**
  - Stateless except for Redis cache
  - Redis required for cache sharing across replicas
  - Memory: buffer pools reduce GC, each request ~3x input size
  - CPU: concurrent processing configurable via `PRIVACY_CONCURRENCY_*`
- **Security:**
  - No hardcoded credentials
  - Use `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` for Redis auth
  - Input size limits (default 10MB, enforced by leakspok)

### 7. `docs/configuration.md`

**Purpose:** Complete env var reference organized by domain.

**Sections:**

- **Loading:** Service reads from environment variables via `caarlos0/env`. Supports `.env` files when using `task run` (Taskfile `dotenv` directive).
- **Server:**

  | Variable | Default | Description |
  |----------|---------|-------------|
  | `PORT` | `8080` | HTTP server port |
  | `HOST` | `0.0.0.0` | HTTP server host |
  | `GRACEFUL_SHUTDOWN_TIMEOUT` | `30s` | Max time to wait for in-flight requests during shutdown |
  | `MAX_BATCH_SIZE` | `100` | Max items per batch request |
  | `LOG_LEVEL` | `INFO` | `DEBUG`, `INFO`, `WARN`, `ERROR` |
  | `SERVICE_NAME` | `""` | OTel service name |
  | `PATTERN_MONITORING_ENABLED` | `false` | Enable leakspok pattern monitoring |

- **Redis / Cache (`PRIVACY_CACHE_*`):**

  | Variable | Default | Description |
  |----------|---------|-------------|
  | `PRIVACY_CACHE_ENABLED` | `false` | Enable rule matching cache |
  | `PRIVACY_CACHE_TTL` | `1h` | Cache entry TTL |
  | `PRIVACY_CACHE_REDIS_ADDR` | `""` | Valkey/Redis address (`host:port`) |
  | `PRIVACY_CACHE_REDIS_DISABLE_CLUSTER` | `false` | Use standalone client instead of cluster |
  | `PRIVACY_CACHE_REDIS_DIAL_TIMEOUT` | `0` | Connection dial timeout |
  | `PRIVACY_CACHE_REDIS_READ_TIMEOUT` | `0` | Socket read timeout |
  | `PRIVACY_CACHE_REDIS_WRITE_TIMEOUT` | `0` | Socket write timeout |
  | `PRIVACY_CACHE_REDIS_POOL_SIZE` | `0` | Max connections per CPU |
  | `PRIVACY_CACHE_REDIS_MIN_IDLE_CONNS` | `0` | Min idle connections |
  | `PRIVACY_CACHE_METRICS` | `true` | Enable cache Prometheus metrics |
  | `PRIVACY_CACHE_DISABLE_IN_MEMORY` | `false` | Disable in-memory server-assisted client-side caching |
  | `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` | `""` | Redis auth password |

- **Concurrency (`PRIVACY_CONCURRENCY_*`):**

  | Variable | Default | Description |
  |----------|---------|-------------|
  | `PRIVACY_CONCURRENCY_ENABLED` | `false` | Enable concurrent processing |
  | `PRIVACY_CONCURRENCY_TOKEN_PROCESSING` | `false` | Parallel token evaluation |
  | `PRIVACY_CONCURRENCY_RULE_PROCESSING` | `false` | Parallel rule evaluation |
  | `PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE` | `0` | Rule runner goroutine pool size |
  | `PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE` | `0` | Token goroutine pool size |
  | `PRIVACY_CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT` | `10s` | Idle goroutine reclamation timeout |

- **OpenTelemetry (`OTEL_*`):**

  | Variable | Default | Description |
  |----------|---------|-------------|
  | `OTEL_ENABLED` | `false` | Enable OpenTelemetry |
  | `OTEL_EXPORTER_ADDR` | `localhost:4317` | OTel exporter address |

### 8. `docs/architecture.md`

**Purpose:** Visual and textual reference for how the service works internally, with emphasis on performance-oriented design choices.

**Sections:**

- **Design philosophy:** The anonymizer is built to handle large text payloads (e.g., AI prompts, log streams) with minimal overhead. Key design decisions:
  - **Byte-level processing:** The entire pipeline uses `[]byte` — no string conversions to avoid unnecessary allocations and copying. `ByteString` type enables zero-copy JSON marshaling.
  - **Buffer pooling:** `sync.Pool` reuses buffers and entity maps across requests, reducing GC pressure under load.
  - **Pluggable caching:** Rule matching results are cached. In-memory client-side caching avoids network round-trips for hot paths. Optional Redis/Valkey backends share state across replicas.
  - **Optional concurrency:** Token-level and rule-level parallelism can be enabled independently to saturate multiple CPU cores for large payloads.
- **Mermaid diagram:** Request flow from HTTP → middleware chain → content-type dispatch → handler → privacy service → rule builder → leakspok ByteAnalyzer → response. Plugin hook point highlighted. Cache and concurrency layers shown as optional branches.
- **Component descriptions:**
  - `AnonymizerServer` (`pkg/server/`): application builder, functional options, plugin registration, handler assembly
  - HTTP handler (`internal/handler/`): content-type dispatch, JSON/text-plain parsing, error responses, `sync.Pool` buffer reuse
  - Privacy service (`pkg/privacy/`): `Service.Anonymize()` orchestrates rule building + leakspok execution
  - Rule builder (`pkg/privacy/rule_builder.go`): converts `PrivacySettings` to `[]analyzer.Rule`
  - leakspok `ByteAnalyzer`: the PII detection and anonymization engine — works exclusively with `[]byte`
- **Plugin system:** `MiddlewareRegistrar` interface, `CoreServices`, context-based rule injection via `WithRules`/`RulesFromContext`
- **Cache layer:** In-memory client-side caching → Redis/Valkey for shared state (optional). Server-assisted client-side caching to minimize network overhead.
- **Concurrency layer:** Token-level parallelism and rule-level parallelism (both optional, configurable). Goroutine pools with configurable pool size and idle reclamation to avoid unbounded goroutine growth on large texts.
- **Package map:** Table showing each package, its responsibility, and its public API surface

### 9. `docs/observability.md`

**Purpose:** Guide for monitoring and debugging the service in production.

**Sections:**

- **Logging:**
  - Uses `log/slog` (Go standard library structured logging)
  - JSON format in production, text format in development
  - Log levels: `DEBUG`, `INFO`, `WARN`, `ERROR`
  - What gets logged: request start/end, anonymization results, cache hits/misses, errors, plugin events
- **Metrics:**
  - Exposed at `/metrics` (Prometheus format)
  - Available metrics: request count, request duration histogram, anonymization duration, cache hit/miss, active requests gauge, error count by code
  - Cache metrics (when `PRIVACY_CACHE_METRICS=true`): cache operations, hit rate, invalidation events
- **Tracing:**
  - OpenTelemetry via `OTEL_ENABLED=true`
  - Exporter address via `OTEL_EXPORTER_ADDR`
  - Span names: `anonymize`, `anonymize_batch`, `anonymize_json`, `anonymize_text`
  - Span attributes: entity count, content type, detected entities, error codes
- **Health check:**
  - `GET /health` returns `{"status":"healthy"}` (200)
  - Does not check Redis connectivity (simple liveness probe)
- **Access logging:** Every request logs method, path, status code, duration, and content type via chi middleware

### 10. `docs/openapi.yaml`

**Purpose:** Machine-readable API specification for tooling and client generation.

**Specification:** OpenAPI 3.1.0

**Contents:**
- Server URL, description
- `POST /api/v1/anonymize` — JSON request/response schema, text/plain headers, error responses
- `POST /api/v1/anonymize/batch` — JSON array request/response schema, error responses
- `GET /health` — response schema
- Schemas: `AnonymizeRequest`, `AnonymizeResponse`, `PrivacySettings`, `EntitySettings`, `RedactionSettings`, `MaskSettings`, `ExceptionSettings`, `MatchSettings`, `AnonymizeBatchResponse`, `ErrorResponse`
- Examples for both JSON and text/plain modes
- Content negotiation documented via `requestBody.content`

### `docker-compose.yml` (Project Root)

```yaml
version: "3.8"
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  anonymizer:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - LOG_LEVEL=INFO
      - PRIVACY_CACHE_ENABLED=true
      - PRIVACY_CACHE_REDIS_ADDR=redis:6379
    depends_on:
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 3
```

### `README.md` Updates

Current README duplicates entity tables, error codes, and env vars. After this work, README should:

- Keep: intro paragraph, feature list (bullet list from current "Features" section), architecture diagram (tree structure)
- Add to feature list: "**High performance** — byte-level processing, buffer pooling, and optional concurrency for low-latency anonymization of large text payloads such as AI prompts"
- Keep: quick start snippet pointing to `docs/getting-started.md`
- Trim: remove full env var table → link to `docs/configuration.md`
- Trim: remove entity table → link to `docs/entities.md`
- Trim: remove exception operators table → link to `docs/anonymize.md#exceptions`
- Keep: "Using as a Library" and "Plugin System" code snippets (they're succinct)
- Keep: testing commands, dependencies, license

### Existing File Updates

**`docs/anonymize.md`:**
- Remove "Supported Entities" section → replace with: `For the full entity reference, see [entities.md](./entities.md).`
- Remove "Error Responses" table → replace with: `For the complete error code reference, see [errors.md](./errors.md).`
- Keep: request format, response format, exceptions (match operators), examples — all of these are specific to this endpoint

**`docs/batch.md`:**
- Remove "Supported Entities" section → link to `entities.md`
- Remove "Error Responses" table → link to `errors.md`
- Keep: request format, response format, limits, examples

**`docs/plugins.md`:**
- Fix stale package paths: `pkg/anonymizer` → `pkg/server`, `anonymizer.NewFromConfig` → `server.NewFromConfig`
- Fix stale import paths: `anonymizer-service-v2/pkg/anonymizer` → `github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server`
- Fix stale import path: `anonymizer-service-v2/pkg/privacy` → `github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy`
- Fix stale import path: `anonymizer-service-v2/pkg/anonymizer` in function signatures to `github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server`
- Add cross-link to `architecture.md`

## Cross-Linking Map

```
getting-started.md → entities.md, content-negotiation.md, configuration.md, plugins.md
entities.md        → redaction.md
redaction.md       → anonymize.md, batch.md
content-negotiation.md → anonymize.md, batch.md, plugins.md
errors.md          → anonymize.md, batch.md
deployment.md      → configuration.md, architecture.md
configuration.md   → deployment.md, entities.md
architecture.md    → plugins.md, observability.md
observability.md   → configuration.md, deployment.md
anonymize.md       → entities.md, errors.md, content-negotiation.md, batch.md
batch.md           → entities.md, errors.md, anonymize.md
plugins.md         → architecture.md, content-negotiation.md
README.md          → getting-started.md, entities.md, configuration.md
openapi.yaml       (standalone, machine-readable)
```

## Verification

- [ ] All Markdown links resolve (no broken references)
- [ ] No duplicated content between files (entity table in one place, error table in one place)
- [ ] All code examples use correct public module path (`github.com/Prosus-Cyber-Xchange/anonymizer`)
- [ ] docker-compose.yml builds and serves a healthy container
- [ ] Every `curl` example works against a locally running service
- [ ] OpenAPI spec validates (`swagger-cli validate docs/openapi.yaml` or equivalent)
- [ ] Mermaid diagram renders correctly in GitHub's Markdown viewer
