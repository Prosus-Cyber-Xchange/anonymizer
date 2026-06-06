# Anonymizer

A REST API service and embeddable Go library that detects and anonymizes personally identifiable information (PII) using the [leakspok](https://github.com/Prosus-Cyber-Xchange/leakspok) library.

## Features

- **Inline privacy rules** — supply anonymization settings directly in the request body
- **Header-based rules** — text/plain requests use `X-Anonymize-*` headers
- **Batch processing** — anonymize multiple texts in a single request
- **Exception handling** — define patterns that should not be anonymized
- **Redaction and masking** — replace PII entirely or partially mask it
- **Valkey/Redis caching** — rule matching results cached via server-assisted client-side caching
- **Plugin system** — inject middleware into the HTTP chain at compile time
- **Embeddable library** — import as a Go package and embed in any application
- **Structured logging** — JSON logging via `log/slog`

## Architecture

```
├── pkg/
│   ├── anonymizer/         # Public library — App builder, options, plugin interfaces
│   ├── privacy/            # Core anonymization service and rule builder
│   ├── config/             # Environment variable loading
│   └── context/            # Context key/values
├── internal/
│   ├── handler/            # HTTP handlers and middleware
│   └── monitoring/         # Prometheus metrics
├── cmd/server/             # Application entry point
├── e2e/                    # End-to-end tests (specification + driver pattern)
│   ├── driver/             # HTTP driver
│   └── specifications/     # Protocol-agnostic test specs
├── examples/plugin/        # Example plugin implementation
├── vendor/                 # Vendored dependencies
└── data/                   # Sample data
```

## Quick Start

```bash
# Copy the environment file and adjust as needed
cp .env.example .env

# Start Redis and run the service
task run
```

This starts a Redis container, loads `.env`, and runs the service. To stop Redis:

```bash
task redis:down
```

## Configuration

Copy `.env.example` to `.env`. The `task run` command loads it automatically via Taskfile's `dotenv` directive. All cache/concurrency/OTel variables are available to the Dockerfile and production workloads.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `HOST` | HTTP server host | `0.0.0.0` |
| `LOG_LEVEL` | Logging level (`DEBUG`, `INFO`, `WARN`, `ERROR`) | `INFO` |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | Server shutdown timeout | `30s` |
| `MAX_BATCH_SIZE` | Max items per batch request | `100` |
| `PATTERN_MONITORING_ENABLED` | Enable leakspok pattern monitoring | `false` |
| `PRIVACY_CACHE_ENABLED` | Enable rule matching cache | `false` |
| `PRIVACY_CACHE_TTL` | Cache entry TTL | `1h` |
| `PRIVACY_CACHE_REDIS_ADDR` | Valkey/Redis address (`host:port`) | `""` |
| `PRIVACY_CACHE_REDIS_DISABLE_CLUSTER` | Use standalone client instead of cluster | `false` |
| `PRIVACY_CACHE_REDIS_POOL_SIZE` | Max connections per CPU | `0` |
| `PRIVACY_CACHE_REDIS_MIN_IDLE_CONNS` | Min idle connections | `0` |
| `PRIVACY_CACHE_REDIS_DIAL_TIMEOUT` | Connection dial timeout | `0` |
| `PRIVACY_CACHE_REDIS_READ_TIMEOUT` | Socket read timeout | `0` |
| `PRIVACY_CACHE_REDIS_WRITE_TIMEOUT` | Socket write timeout | `0` |
| `PRIVACY_CACHE_DISABLE_IN_MEMORY` | Disable server-assisted client-side caching | `false` |
| `PRIVACY_CACHE_METRICS` | Enable cache Prometheus metrics | `true` |
| `PRIVACY_CONCURRENCY_ENABLED` | Enable concurrent processing | `false` |
| `PRIVACY_CONCURRENCY_TOKEN_PROCESSING` | Parallel token evaluation | `false` |
| `PRIVACY_CONCURRENCY_RULE_PROCESSING` | Parallel rule evaluation | `false` |
| `PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE` | Rule runner goroutine pool size | `0` |
| `PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE` | Token goroutine pool size | `0` |
| `PRIVACY_CONCURRENCY_MAX_GOROUTINE_IDLE_TIMEOUT` | Idle goroutine reclamation timeout | `10s` |
| `OTEL_ENABLED` | Enable OpenTelemetry | `false` |
| `OTEL_EXPORTER_ADDR` | OTel exporter address | `localhost:4317` |
| `REDIS_ANONYMIZER_SERVICE_V2_CACHE_TOKEN` | Redis auth password | `""` |

## API

### `POST /api/v1/anonymize`

Privacy rules provided inline in the JSON body, or via headers for text/plain.

**JSON request:**
```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Contact john@example.com, CPF: 123.456.789-09",
    "settings": {
      "entities": [
        { "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } },
        { "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF>" } }
      ]
    }
  }'
```

**JSON response:**
```json
{
  "anonymized_text": "Contact <EMAIL>, CPF: <CPF>",
  "detected_entities": ["CPF_NUMBER", "EMAIL"],
  "anonymized_entities": ["CPF_NUMBER", "EMAIL"]
}
```

**text/plain request with headers:**
```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Anonymize-Entities: EMAIL" \
  -H "X-Anonymize-Placeholder: <REDACTED>" \
  -d "Contact john@example.com"
```

**text/plain response headers:**
```
X-Anonymize-Detected-Entities: EMAIL
X-Anonymize-Anonymized-Entities: EMAIL
```

### `POST /api/v1/anonymize/batch`

```bash
curl -X POST http://localhost:8080/api/v1/anonymize/batch \
  -H "Content-Type: application/json" \
  -d '[
    {
      "text": "Email: john@example.com",
      "settings": { "entities": [{ "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } }] }
    },
    {
      "text": "CPF: 123.456.789-09",
      "settings": { "entities": [{ "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF>" } }] }
    }
  ]'
```

### `GET /health`

```bash
curl http://localhost:8080/health
```

## Supported Entities

| Entity | Description |
|--------|-------------|
| `EMAIL` | Email addresses |
| `CPF_NUMBER` | Brazilian CPF |
| `CNPJ_NUMBER` | Brazilian CNPJ |
| `IP` / `IP_ADDRESS` | IPv4 and IPv6 |
| `IPV4` | IPv4 only |
| `IPV6` | IPv6 only |
| `CREDIT_CARD` | Credit card numbers |
| `PHONE` | Phone numbers |
| `LINK` / `URL` | URLs |
| `SSN` | US Social Security Numbers |
| `ADDRESS` | Street addresses |
| `BANK_INFO` | Banking information (IBAN) |
| `UUID` | UUIDs and GUIDs |

### Exception Operators

| Operator | Description |
|----------|-------------|
| `equal` | Exact match (case-sensitive) |
| `ignoreCaseEqual` | Exact match (case-insensitive) |
| `startsWith` | Prefix match |
| `endsWith` | Suffix match |

## Using as a Library

```go
import "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/anonymizer"

app, err := anonymizer.NewFromConfig(ctx)
if err != nil {
    log.Fatal(err)
}

// Embed in your existing server
mux.Handle("/anon/", http.StripPrefix("/anon", app.Handler()))

// Or run standalone
app.ListenAndServe(ctx)
```

### Plugin System

Plugins implement `MiddlewareRegistrar` to inject middleware into the HTTP chain:

```go
type myPlugin struct{}

func (p *myPlugin) Middleware(svc anonymizer.CoreServices) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // use svc.Logger, svc.ByteAnalyzer
            ctx := context.WithValue(r.Context(), "my_key", "value")
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

Register with `WithPlugin`:

```go
app, _ := anonymizer.NewFromConfig(ctx, anonymizer.WithPlugin(&myPlugin{}))
```

## Testing

```bash
# All tests
task test

# Unit tests only (skips e2e)
task test:unit

# E2E tests (includes Redis with testcontainers)
task test:e2e
```

## Dependencies

- [leakspok](https://github.com/Prosus-Cyber-Xchange/leakspok) — PII detection and anonymization
- [chi](https://github.com/go-chi/chi) — HTTP router
- [valkey-go](https://github.com/valkey-io/valkey-go) — Valkey/Redis client with server-assisted client-side caching
- [prometheus/client_golang](https://github.com/prometheus/client_golang) — Metrics
- [caarlos0/env](https://github.com/caarlos0/env) — Environment variable parsing
- [testcontainers-go](https://github.com/testcontainers/testcontainers-go) — E2E test infrastructure

## License

[Apache 2.0](LICENSE)
