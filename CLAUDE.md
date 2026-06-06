# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a REST API service and embeddable Go library that anonymizes PII (Personally Identifiable Information) using the [leakspok](https://github.com/Prosus-Cyber-Xchange/leakspok) library. It is an open-source version without internal iFood platform integrations.

## Common Commands

### Development
```bash
# Run locally with hot reload
task run

# Build for production (Linux static binary)
task build

# Format code
task format

# Lint code
task lint
```

### Testing
```bash
# Run all tests
task test

# Run only unit tests (using -short flag)
task test/unit

# Run only e2e tests
task test/e2e

# Run specific test
go test -v -run TestName ./privacy/...
```

### Code Generation
```bash
# Generate mocks and other code
task gen
```

### Running the Service
```bash
# Required environment variables
export PORT=8080
export GRACEFUL_SHUTDOWN_TIMEOUT=30s

# Optional (with defaults)
export LOG_LEVEL=INFO

go run cmd/server/main.go
```

## Architecture & Key Concepts

### Package Organization (Domain-Driven)

- `anonymizer/` — Public library package: `App` builder, plugin interfaces, functional options
- `privacy/` — Core business logic (anonymization service, rule building, settings)
- `config/` — Infrastructure concerns (env vars)
- `internal/handler/` — HTTP layer (handlers, routing)
- `internal/monitoring/` — Metrics registry
- `cmd/server/` — Application entry point (thin wrapper using `anonymizer.New()`)

**AVOID** generic names like `utils`, `common`, `helpers`. Name packages after their domain responsibility.

### Request Flow

1. **HTTP Handler** (`internal/handler/handler.go`) decodes the JSON request body containing inline privacy settings
2. **Privacy Service** (`privacy/service.go`) calls the `PrivacyRulesLoader` interface to build rules from settings
3. **Service** applies anonymization using leakspok's `ByteAnalyzer` and returns results

### Plugin System

The `anonymizer` package exposes two plugin interfaces:

- **`RouteRegistrar`**: plugins implementing this add HTTP endpoints under `/api/v1`
- **`RulesLoaderProvider`**: plugins implementing this supply a custom `PrivacyRulesLoader` (e.g., database-backed)

Plugins are registered via `anonymizer.WithPlugin(p any)` and wired at startup via type assertion. A plugin may implement any combination of these interfaces.

```go
app, err := anonymizer.New(anonymizer.Config{Port: 8080},
    anonymizer.WithPlugin(myPlugin),
    anonymizer.WithLogger(logger),
)
```

### Interface Pattern: Declare Where Used

**IMPORTANT**: The `PrivacyRulesLoader` interface is declared in `privacy/service.go` (where it's used), NOT in the package that implements it. This follows the dependency inversion principle.

```go
// privacy/service.go
type PrivacyRulesLoader interface {
    Load(serviceName string) ([]analyzer.Rule, error)
}
```

When adding new loaders (e.g., database, Redis), implement the `PrivacyRulesLoader` interface from the `privacy` package and register them via the `RulesLoaderProvider` plugin interface.

### Byte-Based Processing

**CRITICAL**: The service uses `leakspok.ByteAnalyzer` and works with `[]byte` throughout the request pipeline. Do NOT convert to strings unnecessarily. The `AnonymizeInput` and `AnonymizeOutput` structs use `[]byte` for the body field.

### Constants for Type Safety

When working with entities or match operators, use the constants defined in `privacy/rule_builder.go`:

```go
// Match operators
privacy.MatchOperatorEqual
privacy.MatchOperatorIgnoreCaseEqual
privacy.MatchOperatorStartsWith
privacy.MatchOperatorEndsWith

// Entity types (from leakspok)
pattern.EntityEmail
pattern.EntityCPF
pattern.EntityCNPJ
pattern.EntityIPAddress
// etc.
```

### Dependencies

- **foodsec-go-sdk**: Provides HTTP server, logging with `log/slog`, config loading
  - Use `entrypoint.Do()` for application lifecycle
  - Use `logging.NewSLogger()` for structured logging
- **leakspok**: PII detection and anonymization
  - Use `analyzer.ByteAnalyzer` (not StringAnalyzer)
  - Use entity constants from `pattern` package
- **chi**: HTTP routing
- **env**: Environment variable parsing (via foodsec-go-sdk wrapper)

## Code Standards (Project-Specific)

### Interfaces
- Declare interfaces **where they are used**, not where implemented
- Keep interfaces small and focused (interface segregation principle)

### Composition
- Prefer composition over inheritance
- Use struct embedding sparingly and intentionally

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to load rules for %s: %w", name, err)`
- Use `entrypoint.ExitOnError()` for fatal errors during startup

### Testing
- Use `testify` for assertions
- Use `go.uber.org/mock` for mocks (generated via `go generate`)
- Mock interfaces, not concrete types

### Validation
- Validate privacy settings in `config.ValidateConfig()`
- Operator names are case-insensitive but normalized to lowercase internally

## API Contract

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/anonymize` | General — inline privacy settings in JSON body |
| `POST` | `/api/v1/anonymize/batch` | Batch — array of items, each with its own settings |

### `POST /api/v1/anonymize`
```json
{
  "text": "raw text to anonymize",
  "settings": {
    "entities": [
      { "name": "EMAIL", "redaction": { "replacement": "<EMAIL_REDACTED>" } }
    ]
  }
}
```

Response:
```json
{
  "anonymized_text": "anonymized text",
  "detected_entities": ["EMAIL"],
  "anonymized_entities": ["EMAIL"]
}
```

### `POST /api/v1/anonymize/batch`

Array of the same request shape as `/api/v1/anonymize`. Max items controlled by `MAX_BATCH_SIZE` env var (default: 100). Fails fast on the first invalid or erroring item.

```json
[
  { "text": "...", "settings": { ... } },
  { "text": "...", "settings": { ... } }
]
```

Response: array of the same response shape as `/api/v1/anonymize`.

### Supported Entities
EMAIL, CPF_NUMBER, CNPJ_NUMBER, IP / IP_ADDRESS, IPV4, IPV6, CREDIT_CARD, PHONE, LINK / URL, SSN, ADDRESS, BANK_INFO, UUID

(See `privacy.getMatcherForEntity()` for the complete mapping)

## Important Implementation Notes

### Exception Matching
Exceptions use the same pattern matching system as leakspok. When building exception matchers, wrap them in a `PatternMatcher` with entity "EXCEPTION" (this is just a placeholder entity name for exceptions).

## Active Technologies
- Go 1.25.0 + leakspok (PII detection), chi v5 (HTTP routing), foodsec-go-sdk (server lifecycle, logging, config), Datadog (tracing/profiling), tally (metrics), testify + uber-go/mock (testing)
- Optional Redis (leakspok cache)
