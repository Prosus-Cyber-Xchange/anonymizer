package specifications

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AnonymizeJSON_CacheConsistency(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		body := `{"text":"Contact us at john@example.com","settings":{"entities":[{"name":"EMAIL","redaction":{"replacement":"<EMAIL_REDACTED>"}}]}}`
		headers := jsonHeaders()

		resp1, err := api.Anonymize(context.Background(), body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp1.StatusCode)
		result1 := decodeJSON(t, resp1.Body)

		resp2, err := api.Anonymize(context.Background(), body, headers)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp2.StatusCode)
		result2 := decodeJSON(t, resp2.Body)

		assert.Equal(t, result1.AnonymizedText, result2.AnonymizedText,
			"cached result should match the first response")
		assert.Equal(t, result1.DetectedEntities, result2.DetectedEntities,
			"detected entities should be consistent across cached requests")
		assert.Equal(t, result1.AnonymizedEntities, result2.AnonymizedEntities,
			"anonymized entities should be consistent across cached requests")
	}
}