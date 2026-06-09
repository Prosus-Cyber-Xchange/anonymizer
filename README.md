# Anonymizer

A REST API service and embeddable Go library that detects and anonymizes personally identifiable information (PII) using the [leakspok](https://github.com/Prosus-Cyber-Xchange/leakspok) library.

Built for **high performance** — byte-level processing, buffer pooling, and optional concurrency deliver low-latency anonymization of large text payloads such as AI prompts.

**Documentation site:** [Prosus-Cyber-Xchange.github.io/anonymizer](https://Prosus-Cyber-Xchange.github.io/anonymizer)

## Features

- **Inline privacy rules** — supply anonymization settings directly in the request body
- **Header-based rules** — text/plain requests use `X-Anonymize-*` headers
- **Batch processing** — anonymize multiple texts in a single request
- **Exception handling** — define patterns that should not be anonymized
- **Redaction and masking** — replace PII entirely or partially mask it
- **High performance** — byte-level processing, buffer pooling, and optional concurrency for large payloads
- **Valkey/Redis caching** — rule matching results cached via server-assisted client-side caching
- **Plugin system** — inject middleware into the HTTP chain at compile time
- **Embeddable library** — import as a Go package and embed in any application
- **Structured logging** — JSON logging via `log/slog`

## Quick Start

See the [Getting Started Guide](./docs/content/getting-started.md) for full Docker and manual setup instructions.

```bash
docker compose up
curl http://localhost:8080/health
```

## Documentation

- [Getting Started](./docs/content/getting-started.md) — installation, first request, library usage
- [Entity Reference](./docs/content/entities.md) — all supported PII types
- [Redaction Strategies](./docs/content/redaction.md) — redact and mask configuration
- [Content Negotiation](./docs/content/content-negotiation.md) — JSON vs text/plain modes
- [Error Reference](./docs/content/errors.md) — complete error code reference
- [Configuration Reference](./docs/content/configuration.md) — all environment variables
- [Architecture](./docs/content/architecture.md) — design philosophy, component map, request flow
- [Observability Guide](./docs/content/observability.md) — metrics, tracing, logging
- [Deployment Guide](./docs/content/deployment.md) — Docker, Kubernetes, scaling
- [Plugin Developer Guide](./docs/content/plugins.md) — extending the service with custom middleware
- [API Specification](./docs/content/openapi.yaml) — OpenAPI 3.1 spec

## API

### `POST /api/v1/anonymize`

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

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Anonymize-Entities: EMAIL" \
  -H "X-Anonymize-Placeholder: <REDACTED>" \
  -d "Contact john@example.com"
```

### `POST /api/v1/anonymize/batch`

```bash
curl -X POST http://localhost:8080/api/v1/anonymize/batch \
  -H "Content-Type: application/json" \
  -d '[
    { "text": "Email: john@example.com", "settings": { "entities": [{ "name": "EMAIL", "redaction": { "replacement": "<EMAIL>" } }] } },
    { "text": "CPF: 123.456.789-09", "settings": { "entities": [{ "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF>" } }] } }
  ]'
```

## Using as a Library

```go
import "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"

anonymizer, err := server.NewFromConfig(ctx)
if err != nil {
    log.Fatal(err)
}

// Embed in your existing server
mux.Handle("/anon/", http.StripPrefix("/anon", anonymizer.Handler()))

// Or run standalone
anonymizer.ListenAndServe(ctx)
```

### Plugin System

Plugins implement `MiddlewareRegistrar` to inject middleware into the HTTP chain:

```go
type myPlugin struct{}

func (p *myPlugin) Middleware(svc server.CoreServices) func(http.Handler) http.Handler {
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
anonymizer, _ := server.NewFromConfig(ctx, server.WithPlugin(&myPlugin{}))
```

See the [Plugin Developer Guide](./docs/content/plugins.md) for a complete walkthrough with examples.

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

## Contact

This project is maintained by iFood's AI Security team:

| Name | Email |
|------|-------|
| Caio Cavalcante | caio.cavalcante@ifood.com.br |
| Emanuel Valente | emanuel.valente@ifood.com.br |
| José Almas | jose.almas@ifood.com.br |
| Michelle Mesquita | michelle.mesquita@ifood.com.br |

## License

[Apache 2.0](LICENSE)
