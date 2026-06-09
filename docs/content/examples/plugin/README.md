# Service Rules Plugin Example

This example demonstrates how to embed the Anonymizer as a Go library and register a plugin that injects per-service privacy rules via HTTP headers.

## Architecture

```
HTTP Request                     Anonymizer Service
  │                                    │
  ├─POST /api/v1/anonymize────────────►│
  │ Header: X-Service-Name: email      │
  │                                    ├── Middleware reads X-Service-Name
  │                                    ├── Looks up rules for service
  │                                    ├── Injects rules into context
  │                                    ├── Anonymizer processes with rules
  │                                    │
  │◄───anonymized response─────────────┤
```

## How it works

1. **`ServiceRulesPlugin`** implements `server.MiddlewareRegistrar` — the plugin interface that lets you inject HTTP middleware.
2. On each request, the middleware reads the `X-Service-Name` header.
3. If the header matches a known service, it builds privacy rules from pre-configured settings and injects them into the request context via `server.WithRules`.
4. The anonymizer then uses those context rules for PII detection.

## File overview

| File | Purpose |
|------|---------|
| `main.go` | Entry point — boots the server with the plugin registered |
| `header_plugin.go` | Plugin implementation — middleware, rule lookup, context injection |

## Key interfaces

```go
// MiddlewareRegistrar is declared in pkg/server (where plugins are consumed).
type MiddlewareRegistrar interface {
    Middleware(services CoreServices) func(http.Handler) http.Handler
}
```

`CoreServices` provides access to the logger. In a real application, you'd inject a database client or API caller into your plugin's constructor instead of using a hardcoded map.

## Running

```bash
# Start the service
go run .

# Test with a service that has rules configured
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -H "X-Service-Name: email-service" \
  -d '{"text": "Contact john@example.com or use CPF 123.456.789-09"}'

# Test with an unknown service (no rules applied)
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -H "X-Service-Name: unknown-service" \
  -d '{"text": "Contact john@example.com"}'
```

## Adapting for production

- Replace the hardcoded `rulesByService` map with a database or config store lookup.
- Pass dependencies (API client, cache) through the plugin constructor, not globals.
- Add authorization checks before trusting the `X-Service-Name` header.
- See [Plugin Developer Guide](../../plugins.md) for the full plugin API.
