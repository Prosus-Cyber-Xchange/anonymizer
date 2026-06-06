package specifications

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func HealthCheck(api AnonymizerClient) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := api.Health(context.Background())
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", http.Header(resp.Headers).Get("Content-Type"))
	}
}
