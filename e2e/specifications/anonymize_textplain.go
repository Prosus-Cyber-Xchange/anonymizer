package specifications

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AnonymizeTextPlain_Success tests text/plain with X-Anonymize-Entities header.
func AnonymizeTextPlain_Success(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			"Contact us at john@example.com",
			map[string][]string{
				"Content-Type":            {"text/plain"},
				"X-Anonymize-Entities":    {"EMAIL"},
				"X-Anonymize-Placeholder": {"<EMAIL_REDACTED>"},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Body, "<EMAIL_REDACTED>")
		assert.NotContains(t, resp.Body, "john@example.com")
		assert.NotEmpty(t, http.Header(resp.Headers).Get("X-Anonymize-Detected-Entities"))
		assert.NotEmpty(t, http.Header(resp.Headers).Get("X-Anonymize-Anonymized-Entities"))
	}
}

// AnonymizeTextPlain_EntityHeaderWithSpaces tests whitespace trimming in entity list.
// Adapted from TestHandler_Anonymize_EntityFilteringWithSpaces in internal v2.
func AnonymizeTextPlain_EntityHeaderWithSpaces(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			"Email: john@example.com",
			map[string][]string{
				"Content-Type":            {"text/plain"},
				"X-Anonymize-Entities":    {" email "}, // spaces intentional
				"X-Anonymize-Placeholder": {"<REDACTED>"},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, resp.Body, "<REDACTED>")
		assert.NotContains(t, resp.Body, "john@example.com")
	}
}

// AnonymizeTextPlain_MissingEntitiesHeader tests 400 when no entities provided and no context rules.
func AnonymizeTextPlain_MissingEntitiesHeader(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Anonymize(context.Background(),
			"some text",
			map[string][]string{"Content-Type": {"text/plain"}},
		)
		require.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, resp.Body, "X-Anonymize-Entities")
	}
}
