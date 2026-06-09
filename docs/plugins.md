# Plugin Developer Guide

This guide explains how to write plugins for the anonymizer service. Plugins extend the service with custom middleware that can inject privacy rules, perform logging, modify requests, or integrate with external systems.

## MiddlewareRegistrar Interface

The `MiddlewareRegistrar` interface allows a plugin to inject HTTP middleware into the request chain. Middleware runs before the core anonymization handler.

```go
type MiddlewareRegistrar interface {
    Middleware(services CoreServices) func(http.Handler) http.Handler
}
```

**When to use it:**
- Inject rules from an external source (database, config file, API)
- Log anonymization requests to an audit system
- Validate incoming requests before they reach the handler
- Enrich the request context with data needed downstream
- Authenticate or rate-limit clients

## CoreServices Struct

`CoreServices` provides two shared dependencies to middleware:

```go
type CoreServices struct {
    Logger       *slog.Logger
    ByteAnalyzer analyzer.ByteAnalyzer
}
```

- **Logger**: A structured logger (using Go's `log/slog`) for logging events, errors, and debug info
- **ByteAnalyzer**: A PII analyzer from the leakspok library. Use this to detect entities before anonymization if needed

Example usage:

```go
func (p *MyPlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            services.Logger.Info("processing request", slog.String("path", r.URL.Path))
            next.ServeHTTP(w, r)
        })
    }
}
```

## Context Helpers

Two helpers allow middleware to inject and retrieve privacy rules from the request context.

### WithRules: Inject Rules

```go
func WithRules(ctx context.Context, rules []analyzer.Rule) context.Context
```

Injects privacy rules into the context so the handler can use them if no `X-Anonymize-Entities` header is present.

**Example:**

```go
rules, err := privacy.NewRuleBuilder(settings).Build()
if err != nil {
    services.Logger.Error("failed to build rules", slog.String("error", err.Error()))
    next.ServeHTTP(w, r)
    return
}

ctx := server.WithRules(r.Context(), rules)
next.ServeHTTP(w, r.WithContext(ctx))
```

### RulesFromContext: Retrieve Rules

```go
func RulesFromContext(ctx context.Context) ([]analyzer.Rule, bool)
```

Extracts privacy rules from the context if they were previously injected. Returns the rules and a boolean indicating if they were found.

**Example:**

```go
rules, ok := anonymizer.RulesFromContext(r.Context())
if ok {
    log.Printf("Found %d rules in context", len(rules))
}
```

## Rule Precedence

The handler applies this rule precedence (highest to lowest):

1. **Inline Headers** (`X-Anonymize-Entities`, etc.): If the request includes anonymization headers, they take precedence
2. **Context-Injected Rules**: If no headers are present, the handler checks for rules injected by middleware
3. **400 Error**: If neither headers nor context rules are available, the handler returns a 400 error

This design allows:
- Clients to override rules per request via headers
- Middleware to provide default rules for a client or service
- Explicit opt-in requirement (requests must provide rules somehow)

## Writing a Middleware Plugin: Step-by-Step

### Example: Service-Based Rules Plugin

This example demonstrates a plugin that reads a service name from a request header and injects pre-configured rules for that service into the context.

**Step 1: Define the plugin struct**

```go
package myplugin

import (
    "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
    "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"
)

type ServiceRulesPlugin struct {
    // Store rules per service (in production, fetch from a database or API)
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
```

**Step 2: Implement the MiddlewareRegistrar interface**

```go
import (
    "log/slog"
    "net/http"
)

func (p *ServiceRulesPlugin) Middleware(services anonymizer.CoreServices) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Read service name from header
            serviceName := r.Header.Get("X-Service-Name")
            if serviceName == "" {
                // No service header; pass through
                next.ServeHTTP(w, r)
                return
            }

            // Look up rules for the service
            settings, ok := p.rulesByService[serviceName]
            if !ok {
                services.Logger.Warn("unknown service", slog.String("service", serviceName))
                next.ServeHTTP(w, r)
                return
            }

            // Build rules from settings
            rules, err := privacy.NewRuleBuilder(settings).Build()
            if err != nil {
                services.Logger.Error("failed to build rules", slog.String("error", err.Error()))
                next.ServeHTTP(w, r)
                return
            }

            // Inject rules into context and pass to next handler
            ctx := server.WithRules(r.Context(), rules)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// Verify interface compliance at compile time
var _ server.MiddlewareRegistrar = (*ServiceRulesPlugin)(nil)
```

**Step 3: Register the plugin when building the service**

```go
func main() {
    ctx := context.Background()
    logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

    plugin := myplugin.NewServiceRulesPlugin()

    app, err := server.NewFromConfig(ctx,
        server.WithLogger(logger),
        server.WithPlugin(plugin),
    )
    if err != nil {
        log.Fatal(err)
    }

    if err := app.ListenAndServe(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Testing Plugins with httptest

Use Go's standard `net/http/httptest` package to test middleware in isolation.

### Example: Test Context Rule Injection

```go
package myplugin_test

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
    "myplugin"
)

func TestServiceRulesPlugin_InjectsRulesForKnownService(t *testing.T) {
    // Create the plugin
    plugin := myplugin.NewServiceRulesPlugin()

    // Build the service with the plugin
    app, err := server.NewFromConfig(context.Background(),
        server.WithPlugin(plugin),
    )
    require.NoError(t, err)

    h := app.Handler()

    // Create a request with the service name header but NO X-Anonymize-Entities
    body := "Contact email@example.com"
    req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
    req.Header.Set("Content-Type", "text/plain")
    req.Header.Set("X-Service-Name", "email-service")

    // Record the response
    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)

    // Verify the email was anonymized by the plugin-injected rules
    assert.Equal(t, http.StatusOK, rec.Code)
    assert.NotContains(t, rec.Body.String(), "email@example.com")
    assert.Contains(t, rec.Body.String(), "<EMAIL>")
}

func TestServiceRulesPlugin_IgnoresUnknownService(t *testing.T) {
    plugin := myplugin.NewServiceRulesPlugin()
    app, err := server.NewFromConfig(context.Background(),
        server.WithPlugin(plugin),
    )
    require.NoError(t, err)

    h := app.Handler()

    // Request for unknown service, no X-Anonymize-Entities header
    req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("data"))
    req.Header.Set("Content-Type", "text/plain")
    req.Header.Set("X-Service-Name", "unknown-service")

    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)

    // Should return 400 because neither header nor context rules are available
    assert.Equal(t, http.StatusBadRequest, rec.Code)
    assert.Contains(t, rec.Body.String(), "X-Anonymize-Entities")
}

func TestServiceRulesPlugin_HeadersOverrideContext(t *testing.T) {
    plugin := myplugin.NewServiceRulesPlugin()
    app, err := server.NewFromConfig(context.Background(),
        server.WithPlugin(plugin),
    )
    require.NoError(t, err)

    h := app.Handler()

    // Request with both service header AND X-Anonymize-Entities
    // The header should take precedence
    body := "credit card 1234567890123456 and email@example.com"
    req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
    req.Header.Set("Content-Type", "text/plain")
    req.Header.Set("X-Service-Name", "email-service")
    req.Header.Set("X-Anonymize-Entities", "CREDIT_CARD")

    rec := httptest.NewRecorder()
    h.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    // Verify CREDIT_CARD (from header) was anonymized
    assert.NotContains(t, rec.Body.String(), "1234567890123456")
    // Verify EMAIL was NOT anonymized (only CREDIT_CARD from header applies)
    assert.Contains(t, rec.Body.String(), "email@example.com")
}
```

## Running the Example Plugin

The repository includes a complete example plugin demonstrating service-based rule injection.

```bash
cd examples/plugin
export PORT=8080
go run .
```

The service will start on `http://localhost:8080`. Test it:

```bash
# Request with service header (plugin injects rules)
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Service-Name: email-service" \
  -d "Contact me at user@example.com"

# Request with service header + override via X-Anonymize-Entities
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -H "X-Service-Name: email-service" \
  -H "X-Anonymize-Entities: CREDIT_CARD" \
  -d "Card: 1234567890123456"

# Request without service header or X-Anonymize-Entities (fails)
curl -X POST http://localhost:8080/api/v1/anonymize \
  -H "Content-Type: text/plain" \
  -d "Some data"
```

## Best Practices

1. **Verify Interface Compliance at Compile Time**: Use `var _ server.MiddlewareRegistrar = (*YourPlugin)(nil)` to catch mismatches early
2. **Handle Errors Gracefully**: If rule building fails, log the error and pass through (don't break the chain)
3. **Log Context**: Use structured logging with `slog.String()`, `slog.Int()`, etc. to provide context
4. **Cache When Possible**: Build rules once in the plugin constructor if they don't change per request
5. **Respect Precedence**: Don't override rules if headers are present; let the handler decide
6. **Test with httptest**: Use the standard library's httptest to write integration tests
7. **Document Your Headers**: Clearly document which request headers your plugin expects

## See Also

- [`pkg/server/plugin.go`](../pkg/server/plugin.go) - Core plugin interfaces
- [`pkg/server/context.go`](../pkg/server/context.go) - Context helpers
- [`examples/plugin/`](../examples/plugin/) - Complete working example
- [`pkg/privacy/rule_builder.go`](../pkg/privacy/rule_builder.go) - PrivacySettings and rule building
- [architecture.md](architecture.md) - Service architecture and plugin hook points
