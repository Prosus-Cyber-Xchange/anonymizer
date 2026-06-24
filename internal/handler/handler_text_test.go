package handler_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/internal/handler"
	contextpkg "github.com/Prosus-Cyber-Xchange/anonymizer/pkg/context"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_Anonymize_TextPlain_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("Contact us at john@example.com"))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("X-Anonymize-Entities", "EMAIL")
	req.Header.Set("X-Anonymize-Placeholder", "<EMAIL_REDACTED>")
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "<EMAIL_REDACTED>")
	assert.NotContains(t, rec.Body.String(), "john@example.com")
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Detected-Entities"))
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Anonymized-Entities"))
}

func TestHandler_Anonymize_TextPlain_MissingEntities_NoContextRules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("some text"))
	req.Header.Set("Content-Type", "text/plain")
	// No X-Anonymize-Entities header
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-Anonymize-Entities")
}

func TestHandler_Anonymize_UnsupportedContentType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("<xml/>"))
	req.Header.Set("Content-Type", "application/xml")
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Contains(t, rec.Body.String(), "unsupported content type")
}

func TestHandler_Anonymize_NoContentType_DefaultsJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	body := `{"text":"user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader(body))
	// No Content-Type header — should default to JSON
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "REDACTED")
}

func TestHandler_AnonymizeBatch_NonJSON_Returns415(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()

	h.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Contains(t, rec.Body.String(), "batch endpoint only accepts application/json")
}

func TestHandler_Anonymize_TextPlain_WithContextRules(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	h := handler.NewHandler(handler.HandlerConfig{Logger: logger, PrivacyService: privacyService, MaxBatchSize: maxBatchSize})

	// Build rules for injection
	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{Name: "EMAIL", Redaction: &privacy.RedactionSettings{Replacement: "<PLUGIN_REDACTED>"}},
		},
	}
	rules, err := privacy.NewRuleBuilder(settings, privacy.WithGlobalExceptions(nil)).Build()
	require.NoError(t, err)

	// Create request with context-injected rules (no X-Anonymize-Entities header)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize", strings.NewReader("Contact john@example.com for help"))
	req.Header.Set("Content-Type", "text/plain")
	// Inject rules into context
	req = req.WithContext(contextpkg.WithRules(req.Context(), rules))
	rec := httptest.NewRecorder()

	h.Anonymize(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "<PLUGIN_REDACTED>")
	assert.NotContains(t, rec.Body.String(), "john@example.com")
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Detected-Entities"))
	assert.NotEmpty(t, rec.Header().Get("X-Anonymize-Anonymized-Entities"))
}
