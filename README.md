# Anonymizer Service V2 — Community Edition

A REST API service and embeddable Go library that detects and anonymizes personally identifiable information (PII) in strings using the [leakspok](https://github.com/New-Horizons-Team/leakspok) library.

## Features

- **Flexible anonymization**: Support for both redaction and masking strategies
- **Exception handling**: Define patterns that should not be anonymized
- **Inline privacy rules**: Supply anonymization settings directly in the request body — no YAML config required
- **Batch processing**: Anonymize multiple texts in a single request via `/api/v1/anonymize/batch`
- **Plugin system**: Extend the service with custom endpoints and rules loaders via compile-time plugins
- **Embeddable library**: Import the `anonymizer` package and embed the service in any Go application
- **Comprehensive logging**: Built-in access logging and structured logging
- **Health checks**: Health endpoint for monitoring

## Architecture

```
anonymizer-api-v2-ce/
├── anonymizer/             # Public library package — App builder and plugin interfaces
│   ├── app.go              # App struct, New(), Handler(), ListenAndServe()
│   ├── options.go          # Functional options (WithPlugin, WithLogger, etc.)
│   └── plugin.go           # RouteRegistrar and RulesLoaderProvider interfaces
├── privacy/                # Core business logic (public)
│   ├── service.go          # Anonymization service
│   ├── settings.go         # Configuration models
│   └── rule_builder.go     # Converts config to leakspok rules
├── config/                 # Configuration loading (public)
│   └── env.go              # Environment variables
├── internal/
│   └── handler/            # HTTP handlers (internal)
│       ├── handler.go
│       ├── handler_metrics.go
│       └── router.go
├── cmd/server/             # Application entry point
│   └── main.go
```

## API Schema

The service exposes two anonymization endpoints:

| Endpoint | Description |
|----------|-------------|
| `POST /api/v1/anonymize` | General endpoint — privacy rules provided inline in the request body |
| `POST /api/v1/anonymize/batch` | Batch endpoint — process multiple items in a single request |

---

### `POST /api/v1/anonymize`

Privacy rules are provided inline in the JSON request body.

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Contact joe.doe@company.com, CPF: 123.456.789-09",
    "settings": {
      "entities": [
        {
          "name": "EMAIL",
          "redaction": { "replacement": "<EMAIL_REDACTED>" }
        },
        {
          "name": "CPF_NUMBER",
          "redaction": { "replacement": "<CPF_REDACTED>" }
        }
      ]
    }
  }'
```

**Response:**
```json
{
  "anonymized_text": "Contact <EMAIL_REDACTED>, CPF: <CPF_REDACTED>",
  "detected_entities": ["CPF_NUMBER", "EMAIL"],
  "anonymized_entities": ["CPF_NUMBER", "EMAIL"]
}
```

---

### `POST /api/v1/anonymize/batch`

Processes multiple anonymization requests in a single HTTP call. Each item in the array is independent and can have its own privacy settings. The endpoint fails fast: if any item is invalid or fails anonymization, the entire request returns an error.

The maximum number of items per request is controlled by the `MAX_BATCH_SIZE` environment variable (default: `100`).

**Request:**
```bash
curl -X POST http://localhost:8080/api/v1/anonymize/batch \
  -H "Content-Type: application/json" \
  -d '[
    {
      "text": "Contact joe.doe@company.com",
      "settings": {
        "entities": [
          {
            "name": "EMAIL",
            "redaction": { "replacement": "<EMAIL_REDACTED>" }
          }
        ]
      }
    },
    {
      "text": "CPF: 123.456.789-09 and phone: +55 11 99999-9999",
      "settings": {
        "entities": [
          {
            "name": "CPF_NUMBER",
            "redaction": { "replacement": "<CPF_REDACTED>" }
          },
          {
            "name": "PHONE",
            "mask": { "replacement": "*", "maxLength": 4 }
          }
        ]
      }
    }
  ]'
```

**Response:**
```json
[
  {
    "anonymized_text": "Contact <EMAIL_REDACTED>",
    "detected_entities": ["EMAIL"],
    "anonymized_entities": ["EMAIL"]
  },
  {
    "anonymized_text": "CPF: <CPF_REDACTED> and phone: ****9-9999",
    "detected_entities": ["CPF_NUMBER", "PHONE"],
    "anonymized_entities": ["CPF_NUMBER", "PHONE"]
  }
]
```

**Error — batch size exceeded:**
```json
{
  "code": "BATCH_SIZE_EXCEEDED",
  "message": "batch size 150 exceeds maximum allowed size of 100"
}
```

## Using as a Library

Import the `anonymizer` package to embed the service in your own Go application:

```go
import "anonymizer-service-v2/anonymizer"

app, err := anonymizer.New(anonymizer.Config{
    Port:         8080,
    MaxBatchSize: 100,
})
if err != nil {
    log.Fatal(err)
}

// Embed in your existing HTTP server:
mux.Handle("/anon/", http.StripPrefix("/anon", app.Handler()))

// Or run the built-in server:
app.ListenAndServe(ctx)
```

### Plugin System

Plugins extend the service at compile time via two interfaces:

**`RouteRegistrar`** — add custom HTTP endpoints under `/api/v1`:

```go
type myPlugin struct{}

func (p *myPlugin) RegisterRoutes(r chi.Router, svc anonymizer.CoreServices) {
    r.Post("/my-endpoint", func(w http.ResponseWriter, r *http.Request) {
        // use svc.PrivacyService, svc.Logger, svc.ByteAnalyzer
    })
}

app, _ := anonymizer.New(cfg, anonymizer.WithPlugin(&myPlugin{}))
```

**`RulesLoaderProvider`** — supply a custom privacy rules loader (e.g., database-backed):

```go
type myLoader struct{}

func (p *myLoader) RulesLoader(svc anonymizer.CoreServices) (privacy.PrivacyRulesLoader, error) {
    return myDBLoader{db: myDB}, nil
}

app, _ := anonymizer.New(cfg, anonymizer.WithPlugin(&myLoader{}))
```

A plugin may implement both interfaces simultaneously.

## Configuration

### Environment Variables

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | - | Yes |
| `HOST` | HTTP server host | `0.0.0.0` | No |
| `LOG_LEVEL` | Logging level | `INFO` | No |
| `GRACEFUL_SHUTDOWN_TIMEOUT` | Server shutdown timeout | - | Yes |
| `MAX_BATCH_SIZE` | Maximum number of items allowed in a single batch request | `100` | No |

### Supported Entities

- `EMAIL` — Email addresses
- `CPF_NUMBER` — Brazilian CPF
- `CNPJ_NUMBER` — Brazilian CNPJ
- `IP` / `IP_ADDRESS` — IPv4 and IPv6 addresses
- `IPV4` — IPv4 only
- `IPV6` — IPv6 only
- `CREDIT_CARD` — Credit card numbers
- `PHONE` — Phone numbers
- `LINK` / `URL` — URLs
- `SSN` — US Social Security Numbers
- `ADDRESS` — Street addresses
- `BANK_INFO` — Banking information (IBAN)
- `UUID` — UUIDs and GUIDs

### Exception Operators

- `equal` — Exact match (case-sensitive)
- `ignoreCaseEqual` — Exact match (case-insensitive)
- `startsWith` — Prefix match
- `endsWith` — Suffix match

## Running the Service

### Build
```bash
go build -o anonymizer ./cmd/server
```

### Run
```bash
export PORT=8080
export LOG_LEVEL=INFO
export GRACEFUL_SHUTDOWN_TIMEOUT=30s

./anonymizer
```

## Example Usage

### Anonymize endpoint (inline rules)
```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Call me at +55 11 99999-9999",
    "settings": {
      "entities": [
        { "name": "PHONE", "redaction": { "replacement": "<PHONE_REDACTED>" } }
      ]
    }
  }'
```

**Response:**
```json
{
  "anonymized_text": "Call me at <PHONE_REDACTED>",
  "detected_entities": ["PHONE"],
  "anonymized_entities": ["PHONE"]
}
```

### Batch endpoint
```bash
curl -X POST http://localhost:8080/api/v1/anonymize/batch \
  -H "Content-Type: application/json" \
  -d '[
    {
      "text": "Email: joe@example.com",
      "settings": { "entities": [{ "name": "EMAIL", "redaction": { "replacement": "<EMAIL_REDACTED>" } }] }
    },
    {
      "text": "CPF: 123.456.789-09",
      "settings": { "entities": [{ "name": "CPF_NUMBER", "redaction": { "replacement": "<CPF_REDACTED>" } }] }
    }
  ]'
```

**Response:**
```json
[
  {
    "anonymized_text": "Email: <EMAIL_REDACTED>",
    "detected_entities": ["EMAIL"],
    "anonymized_entities": ["EMAIL"]
  },
  {
    "anonymized_text": "CPF: <CPF_REDACTED>",
    "detected_entities": ["CPF_NUMBER"],
    "anonymized_entities": ["CPF_NUMBER"]
  }
]
```

### Health Check
```bash
curl http://localhost:8080/health
```

**Response:**
```
HTTP 200 OK
```

## Testing

```bash
# All tests
go test ./...

# Unit tests only
go test -short ./...
```

## Dependencies

- [leakspok](https://github.com/New-Horizons-Team/leakspok) — PII detection and anonymization
- [foodsec-go-sdk](https://code.ifoodcorp.com.br/ifood/security/libs/go/foodsec-go-sdk) — HTTP server, logging, configuration
- [chi](https://github.com/go-chi/chi) — HTTP router
- [env](https://github.com/caarlos0/env) — Environment variable parsing
- [yaml.v3](https://gopkg.in/yaml.v3) — YAML parsing

## Code Standards

- **Clean Code & SOLID principles**: Maintainable and testable code
- **Domain-driven package naming**: `privacy`, `config`, `handler` (not `utils`, `common`)
- **Interface declaration**: Interfaces declared where used, not where implemented
- **Composition over inheritance**: Extensive use of composition
- **Byte-based processing**: Uses `ByteAnalyzer` to avoid unnecessary string conversions
- **Plugin system**: Compile-time interface composition — no `plugin` package, no dynamic loading

## License

Copyright © 2025 iFood
