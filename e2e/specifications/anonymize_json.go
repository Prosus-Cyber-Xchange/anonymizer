package specifications

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonHeaders is the default header set for JSON requests.
func jsonHeaders() map[string][]string {
	return map[string][]string{"Content-Type": {"application/json"}}
}

type anonymizeJSONResponse struct {
	AnonymizedText     string   `json:"anonymized_text"`
	DetectedEntities   []string `json:"detected_entities"`
	AnonymizedEntities []string `json:"anonymized_entities"`
}

func decodeJSON(t *testing.T, body string) anonymizeJSONResponse {
	t.Helper()
	var r anonymizeJSONResponse
	require.NoError(t, json.Unmarshal([]byte(body), &r))
	return r
}

// AnonymizeJSON_Success tests basic email redaction.
// Adapted from TestHandler_Anonymize_Success in internal v2.
func AnonymizeJSON_Success(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			`{"text":"Contact us at john@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}`,
			jsonHeaders(),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result := decodeJSON(t, resp.Body)
		assert.Contains(t, result.AnonymizedText, "<EMAIL_REDACTED>")
		assert.NotContains(t, result.AnonymizedText, "john@example.com")
		assert.Contains(t, result.DetectedEntities, "EMAIL")
		assert.Contains(t, result.AnonymizedEntities, "EMAIL")
	}
}

// AnonymizeJSON_MultipleEntities tests EMAIL + CPF redaction in one request.
// Adapted from TestHandler_Anonymize_MultipleEntities in internal v2.
func AnonymizeJSON_MultipleEntities(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			`{"text":"Email: john@example.com, CPF: 111.444.777-35","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}},{"name":"CPF_NUMBER","redaction":{"replacement":"<CPF_REDACTED>"}}]}}`,
			jsonHeaders(),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result := decodeJSON(t, resp.Body)
		assert.Contains(t, result.AnonymizedText, "<EMAIL_REDACTED>")
		assert.NotContains(t, result.AnonymizedText, "john@example.com")
		assert.Contains(t, result.DetectedEntities, "EMAIL")
	}
}

// AnonymizeJSON_NoEntitiesDetected tests clean text returns unchanged.
// Adapted from TestHandler_Anonymize_NoEntitiesDetected in internal v2.
func AnonymizeJSON_NoEntitiesDetected(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			`{"text":"No sensitive data here","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}`,
			jsonHeaders(),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result := decodeJSON(t, resp.Body)
		assert.Equal(t, "No sensitive data here", result.AnonymizedText)
		assert.Empty(t, result.DetectedEntities)
		assert.Empty(t, result.AnonymizedEntities)
	}
}

// AnonymizeJSON_InvalidSettings tests empty entities → 400 INVALID_SETTINGS.
func AnonymizeJSON_InvalidSettings(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			`{"text":"hello","settings":{}}`,
			jsonHeaders(),
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, resp.Body, "INVALID_SETTINGS")
	}
}

// AnonymizeJSON_NoContentTypeDefaultsToJSON tests Content-Type omission.
// Adapted from TestHandler_Anonymize_WithoutEntityFilterHeader in internal v2.
func AnonymizeJSON_NoContentTypeDefaultsToJSON(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			`{"text":"user@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`,
			nil, // no headers
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		result := decodeJSON(t, resp.Body)
		assert.Contains(t, result.AnonymizedText, "<REDACTED>")
	}
}
