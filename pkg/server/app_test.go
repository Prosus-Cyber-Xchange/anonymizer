package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromConfig_NoPlugins(t *testing.T) {
	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, app)

	h := app.Handler()
	assert.NotNil(t, h)
}

func TestNewFromConfig_DefaultConfig(t *testing.T) {
	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, app)

	// Handler should be created with defaults
	h := app.Handler()
	assert.NotNil(t, h)
}

func TestHandler_CEEndpointsWork(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)

	h := app.Handler()

	// Test /api/v1/anonymize
	body := `{"text":"Contact user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// JSON encodes < > as \u003c \u003e, so check for the escaped form or just for "REDACTED"
	assert.Contains(t, rec.Body.String(), "REDACTED")
	assert.NotContains(t, rec.Body.String(), "user@example.com")
}

func TestHandler_BatchEndpointWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)

	h := app.Handler()

	body := `[{"text":"user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "REDACTED")
}

func TestHandler_HealthEndpointWorks(t *testing.T) {
	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)

	h := app.Handler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandler_UnknownRouteReturns404(t *testing.T) {
	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)

	h := app.Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/unknown", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// mockMiddlewarePlugin is a test plugin that injects a header into the response.
type mockMiddlewarePlugin struct {
	called bool
}

func (m *mockMiddlewarePlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
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

	app, err := server.NewFromConfig(context.Background(),
		server.WithPlugin(plugin),
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

	app, err := server.NewFromConfig(context.Background(),
		server.WithPlugin(plugin),
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

// contextRulesPlugin injects EMAIL redaction rules into every request context.
type contextRulesPlugin struct{}

func (p *contextRulesPlugin) Middleware(services server.CoreServices) func(http.Handler) http.Handler {
	// Build EMAIL redaction rules once at middleware creation time
	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<PLUGIN_REDACTED>"}},
		},
	}
	rules, _ := privacy.NewRuleBuilder(settings).Build()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := server.WithRules(r.Context(), rules)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestListenAndServe_RespectsContextCancellation(t *testing.T) {
	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should return promptly because context is already cancelled
	err = app.ListenAndServe(ctx)
	// Either nil or a bind error (port 0 might not work) — what matters is it doesn't hang
	_ = err
}

func TestNewFromConfig_ConcurrencyConfigWired(t *testing.T) {
	// Set env vars for concurrency
	t.Setenv("PRIVACY_CONCURRENCY_ENABLED", "true")
	t.Setenv("PRIVACY_CONCURRENCY_TOKEN_PROCESSING", "true")
	t.Setenv("PRIVACY_CONCURRENCY_RULE_PROCESSING", "true")
	t.Setenv("PRIVACY_CONCURRENCY_RULE_RUNNER_POOL_SIZE", "4")
	t.Setenv("PRIVACY_CONCURRENCY_TOKEN_POOL_SIZE", "8")

	app, err := server.NewFromConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, app)

	// Service should initialize without error — verifies config parsing
	h := app.Handler()
	assert.NotNil(t, h)
}
