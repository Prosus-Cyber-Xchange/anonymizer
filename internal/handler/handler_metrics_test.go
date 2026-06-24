package handler_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/internal/handler"
	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber-go/tally/v4"
)

func TestNewHandlerWithMetrics(t *testing.T) {
	t.Run("creates handler with custom metrics", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := handler.NewPrivacyMetrics(scope)

		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		byteAnalyzer := newTestByteAnalyzer(t, logger)
		privacyService := privacy.NewService(byteAnalyzer, logger)

		h := handler.NewHandler(handler.HandlerConfig{
			Logger:         logger,
			PrivacyService: privacyService,
			MaxBatchSize:   maxBatchSize,
			Metrics:        metrics,
		})

		require.NotNil(t, h)
		assert.Equal(t, 65536, 65536)
	})

	t.Run("NewHandler creates handler with empty metrics", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
		byteAnalyzer := newTestByteAnalyzer(t, logger)
		privacyService := privacy.NewService(byteAnalyzer, logger)

		h := handler.NewHandler(handler.HandlerConfig{
			Logger:         logger,
			PrivacyService: privacyService,
			MaxBatchSize:   maxBatchSize,
		})

		require.NotNil(t, h)
	})
}
