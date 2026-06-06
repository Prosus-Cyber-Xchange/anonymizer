package specifications

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type batchItemResponse struct {
	AnonymizedText     string   `json:"anonymized_text"`
	DetectedEntities   []string `json:"detected_entities"`
	AnonymizedEntities []string `json:"anonymized_entities"`
}

// AnonymizeBatch_MultipleItems tests two items with separate settings.
func AnonymizeBatch_MultipleItems(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		body := `[{"text":"Email: john@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}},{"text":"Email: jane@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}]`
		resp, err := api.AnonymizeBatch(context.Background(), body, jsonHeaders())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		var results []batchItemResponse
		require.NoError(t, json.Unmarshal([]byte(resp.Body), &results))
		require.Len(t, results, 2)
		assert.Contains(t, results[0].AnonymizedText, "<EMAIL_REDACTED>")
		assert.NotContains(t, results[0].AnonymizedText, "john@example.com")
		assert.Contains(t, results[1].AnonymizedText, "<EMAIL_REDACTED>")
		assert.NotContains(t, results[1].AnonymizedText, "jane@example.com")
	}
}

// AnonymizeBatch_ExceedsMaxSize tests 400 when batch exceeds MAX_BATCH_SIZE (default 100).
func AnonymizeBatch_ExceedsMaxSize(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		item := `{"text":"hello","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<REDACTED>"}}]}}`
		items := make([]string, 101)
		for i := range items {
			items[i] = item
		}
		body := "[" + strings.Join(items, ",") + "]"
		resp, err := api.AnonymizeBatch(context.Background(), body, jsonHeaders())
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, resp.Body, "BATCH_SIZE_EXCEEDED")
	}
}

// AnonymizeBatch_UnsupportedMediaType tests 415 when batch receives text/plain.
func AnonymizeBatch_UnsupportedMediaType(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.AnonymizeBatch(context.Background(), "plain text",
			map[string][]string{"Content-Type": {"text/plain"}},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusUnsupportedMediaType, resp.StatusCode)
	}
}
