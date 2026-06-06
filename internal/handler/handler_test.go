package handler_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"anonymizer-service-v2/internal/handler"
	"anonymizer-service-v2/pkg/privacy"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	maxBatchSize = 100
)

func newTestByteAnalyzer(t *testing.T, logger *slog.Logger) analyzer.ByteAnalyzer {
	t.Helper()
	a, err := analyzer.MakeByteAnalyzer(context.Background(), logger, analyzer.RunnerOptions{})
	require.NoError(t, err)
	return a
}

// batchResponse mirrors anonymizeResponse for decoding batch results in tests
type batchResponse struct {
	AnonymizedText     json.RawMessage `json:"anonymized_text"`
	DetectedEntities   []string        `json:"detected_entities"`
	AnonymizedEntities []string        `json:"anonymized_entities"`
}

func TestHandler_AnonymizeBatch_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	body := `[{"text":"Contact us at john@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var responses []batchResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responses))
	require.Len(t, responses, 1)

	var anonymizedText string
	require.NoError(t, json.Unmarshal(responses[0].AnonymizedText, &anonymizedText))
	assert.Contains(t, anonymizedText, "<EMAIL_REDACTED>")
	assert.NotContains(t, anonymizedText, "john@example.com")
	assert.Contains(t, responses[0].DetectedEntities, "EMAIL")
	assert.Contains(t, responses[0].AnonymizedEntities, "EMAIL")
}

func TestHandler_AnonymizeBatch_MultipleItems(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	body := `[` +
		`{"text":"Email: john@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}},` +
		`{"text":"Email: jane@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}` +
		`]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var responses []batchResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responses))
	require.Len(t, responses, 2)

	var text0, text1 string
	require.NoError(t, json.Unmarshal(responses[0].AnonymizedText, &text0))
	require.NoError(t, json.Unmarshal(responses[1].AnonymizedText, &text1))

	assert.Contains(t, text0, "<EMAIL_REDACTED>")
	assert.NotContains(t, text0, "john@example.com")
	assert.Contains(t, responses[0].DetectedEntities, "EMAIL")

	assert.Contains(t, text1, "<EMAIL_REDACTED>")
	assert.NotContains(t, text1, "jane@example.com")
	assert.Contains(t, responses[1].DetectedEntities, "EMAIL")
}

func TestHandler_AnonymizeBatch_EmptyBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(`[]`))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var responses []batchResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responses))
	assert.Empty(t, responses)
}

func TestHandler_AnonymizeBatch_ExceedsMaxSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	item := `{"text":"hello","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
	items := make([]string, maxBatchSize+1)
	for i := range items {
		items[i] = item
	}
	body := "[" + strings.Join(items, ",") + "]"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "BATCH_SIZE_EXCEEDED")
}

func TestHandler_AnonymizeBatch_InvalidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(`{not valid json`))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "INVALID_REQUEST")
}

func TestHandler_AnonymizeBatch_InvalidSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	// Empty entities list is invalid per ValidatePrivacyConfig
	body := `[{"text":"hello","settings":{}}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "INVALID_SETTINGS")
}

func TestHandler_AnonymizeBatch_NoPIIDetected(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	privacyService := privacy.NewService(byteAnalyzer, logger)
	handler := handler.NewHandler(logger, privacyService, maxBatchSize)

	body := `[{"text":"No sensitive data here","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}]`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anonymize/batch", strings.NewReader(body))
	rec := httptest.NewRecorder()

	handler.AnonymizeBatch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var responses []batchResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &responses))
	require.Len(t, responses, 1)

	var anonymizedText string
	require.NoError(t, json.Unmarshal(responses[0].AnonymizedText, &anonymizedText))
	assert.Contains(t, anonymizedText, "No sensitive data here")
	assert.Empty(t, responses[0].DetectedEntities)
	assert.Empty(t, responses[0].AnonymizedEntities)
}
