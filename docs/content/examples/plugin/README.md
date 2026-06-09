# Service Rules Plugin Example

This example demonstrates how to embed the Anonymizer as a Go library and register a plugin that injects per-service privacy rules based on an HTTP header.

> Browse the code: [github.com/Prosus-Cyber-Xchange/anonymizer](https://github.com/Prosus-Cyber-Xchange/anonymizer/tree/main/docs/content/examples/plugin/)

> **Files in this folder:**
>
> | File | Purpose |
> |------|---------|
> | `main.go` | Entry point — boots the server with the plugin registered |
> | `header_plugin.go` | Plugin implementation — middleware, rule lookup, context injection |

## Architecture

```
HTTP Request                     Anonymizer Service
  │                                    │
  ├─POST /api/v1/anonymize────────────►│
  │  Header: X-Service-Name: email     │
  │  Body: {"text": "call             │
  │         bob@example.com"}          │
  │                                    ├── Middleware reads X-Service-Name
  │                                    ├── Looks up rules for "email-service"
  │                                    ├── Builds EMAIL + CPF rules
  │                                    ├── Injects rules into context
  │                                    ├── Anonymizer detects "bob@example.com"
  │                                    ├── Replaces with "<EMAIL>"
  │                                    │
  │◄─── {"anonymized_text":           │
  │      "call <EMAIL>"}───────────────┤
```

Each request can target a different service — the plugin looks up the right rules at request time, no restart needed.

## What it demonstrates

The plugin in `header_plugin.go` does three things:

1. **Reads the `X-Service-Name` request header** to determine which service is calling.
2. **Looks up pre-configured privacy rules** for that service (mapping is hardcoded here; in production you'd query a config store).
3. **Injects the rules into the request context** via `server.WithRules(ctx, rules)` so the anonymizer applies them.

## Step-by-step walkthrough

### Step 1 — Prerequisites

- **Go 1.25+** installed
- The anonymizer module available locally or as a dependency

### Step 2 — Understand the plugin interface

The anonymizer accepts anything that implements `server.MiddlewareRegistrar`:

```go
type MiddlewareRegistrar interface {
    Middleware(services CoreServices) func(http.Handler) http.Handler
}
```

`CoreServices` gives the plugin access to the shared logger. Your `Middleware` method returns standard `net/http` middleware — you can do anything a normal HTTP middleware can do.

This example's plugin implements the interface at `header_plugin.go:63`:

```go
var _ server.MiddlewareRegistrar = (*ServiceRulesPlugin)(nil)
```

This line is optional but ensures you get a compile-time error if the interface changes.

### Step 3 — Look at the plugin struct

```go
type ServiceRulesPlugin struct {
    rulesByService map[string]privacy.PrivacySettings
}
```

The `rulesByService` map is a stand-in for a real config source. Each key is a service name, and each value is a `PrivacySettings` struct defining which entities to detect and how to redact them:

```go
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
```

Each `EntitySettings` specifies:

| Field | Purpose |
|-------|---------|
| `Name` | The entity type (`"EMAIL"`, `"CPF_NUMBER"`, `"CREDIT_CARD"`, etc.) |
| `Redaction` | Replaces PII with a fixed string like `"<EMAIL>"` |
| `Mask` | Partially masks PII (e.g. `"j***@example.com"`) |
| `Exceptions` | Patterns to skip (e.g. internal email domains) |

See [entities.md](../../entities.md) for all supported entity names.

### Step 4 — Understand the middleware

The `Middleware` method at `header_plugin.go:33` returns standard HTTP middleware:

```go
func (p *ServiceRulesPlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
```

Inside the handler, the flow is:

**a) Extract the service name:**

```go
serviceName := r.Header.Get("X-Service-Name")
if serviceName == "" {
    next.ServeHTTP(w, r)  // no header → pass through unchanged
    return
}
```

**b) Look up the settings for this service:**

```go
settings, ok := p.rulesByService[serviceName]
if !ok {
    services.Logger.Warn("unknown service", slog.String("service", serviceName))
    next.ServeHTTP(w, r)  // unknown service → pass through
    return
}
```

**c) Build analyzer rules from the settings:**

```go
rules, err := privacy.NewRuleBuilder(settings).Build()
if err != nil {
    services.Logger.Error("failed to build rules", slog.String("error", err.Error()))
    next.ServeHTTP(w, r)
    return
}
```

**d) Inject rules into the request context:**

```go
ctx := server.WithRules(r.Context(), rules)
next.ServeHTTP(w, r.WithContext(ctx))
```

The anonymizer checks the context for rules on every request. If present, they take precedence over the inline `settings` in the JSON body.

### Step 5 — Wire the plugin into main.go

```go
plugin := NewServiceRulesPlugin()

app, err := server.NewFromConfig(ctx,
    server.WithLogger(logger),
    server.WithPlugin(plugin),
)
```

`server.WithPlugin(plugin)` registers the plugin at startup. The `NewFromConfig` function reads environment variables for server configuration (port, timeouts, cache).

### Step 6 — Run the service

```bash
go run .
```

The service starts on port 8080 by default. You'll see:

```
{"level":"INFO","msg":"starting anonymizer with service-rules plugin"}
```

### Step 7 — Test with curl

**Send a request as the "email-service":**

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -H "X-Service-Name: email-service" \
  -d '{"text": "Contact john@example.com or use CPF 123.456.789-09"}'
```

Response:
```json
{
  "anonymized_text": "Contact <EMAIL> or use CPF <CPF>",
  "detected_entities": ["EMAIL", "CPF_NUMBER"],
  "anonymized_entities": ["EMAIL", "CPF_NUMBER"]
}
```

The plugin looked up the rules for `email-service` and applied email + CPF redaction.

**Send a request as the "payment-service":**

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -H "X-Service-Name: payment-service" \
  -d '{"text": "Card 4111-1111-1111-1111 and email bob@example.com"}'
```

Response:
```json
{
  "anonymized_text": "Card <CARD> and email bob@example.com",
  "detected_entities": ["CREDIT_CARD"],
  "anonymized_entities": ["CREDIT_CARD"]
}
```

Only the credit card is redacted — email passes through because `payment-service` doesn't have an EMAIL rule.

**Send a request with no service header:**

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -d '{"text": "Contact john@example.com"}'
```

Response:
```json
{
  "anonymized_text": "Contact john@example.com",
  "detected_entities": [],
  "anonymized_entities": []
}
```

No header, no rules — the text passes through untouched.

**Send a request for an unknown service:**

```bash
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: application/json" \
  -H "X-Service-Name: unknown-service" \
  -d '{"text": "Contact john@example.com"}'
```

The server logs `WARN unknown service` and the request passes through unchanged. No error is returned to the client.

### Step 8 — Stop the service

Press `Ctrl+C`. The server respects OS signals and shuts down gracefully.

## Adapting for production

The hardcoded `rulesByService` map is fine for this example. For production, replace it with a real data source:

- **Database**: Query a `service_rules` table in `NewServiceRulesPlugin` and reload on a schedule.
- **Config file**: Load a YAML/JSON config at startup, watch for changes with `fsnotify`.
- **API client**: Call an external rules service on each request (add caching to avoid per-request latency).

Pass those dependencies through the plugin constructor:

```go
plugin := NewServiceRulesPlugin(dbClient, cacheClient)
```

Add authorization — don't blindly trust the `X-Service-Name` header from an untrusted caller. Validate it against a known service registry or require a signed token.

See [Plugin Developer Guide](../../plugins.md) for the full plugin API, including the `RouteRegistrar` interface for adding custom HTTP endpoints.
