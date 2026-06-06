package privacy_test

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"testing"

	"anonymizer-service-v2/pkg/privacy"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestByteAnalyzer(t *testing.T, logger *slog.Logger) analyzer.ByteAnalyzer {
	t.Helper()
	a, err := analyzer.MakeByteAnalyzer(context.Background(), logger, analyzer.RunnerOptions{})
	require.NoError(t, err)
	return a
}

func TestService_AnonymizeWithRules_EmptyInput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	buffer := bytes.NewBuffer(nil)
	output, err := service.AnonymizeWithRules(context.Background(), nil, []byte{}, buffer)
	require.NoError(t, err)
	assert.Zero(t, output.Details.HasFindings)
}

func TestService_AnonymizeWithRules_NoRules(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	body := []byte("test body")
	buffer := bytes.NewBuffer(nil)
	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{}, body, buffer)
	require.NoError(t, err)
	assert.Zero(t, output.Details.HasFindings)
	assert.Equal(t, body, buffer.Bytes())
}

func TestService_AnonymizeWithRules_WithEmailAnonymization(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	emailRule := analyzer.Rule{
		Name:        "email",
		Description: "Email addresses",
		Matcher:     pattern.EmailMatcher(),
		Settings: analyzer.RuleSettings{
			Strategy: analyzer.REDACT,
			Redact: &analyzer.RedactSettings{
				Placeholder: "<EMAIL_REDACTED>",
			},
		},
	}

	body := []byte("Contact us at john@example.com for support")
	buffer := bytes.NewBuffer(nil)

	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{emailRule}, body, buffer)
	require.NoError(t, err)
	assert.True(t, output.Details.HasFindings)
	assert.Contains(t, string(buffer.Bytes()), "<EMAIL_REDACTED>")
	assert.NotContains(t, string(buffer.Bytes()), "john@example.com")
}

func TestService_AnonymizeWithRules_NoPIIDetected(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	byteAnalyzer := newTestByteAnalyzer(t, logger)
	service := privacy.NewService(byteAnalyzer, logger)

	emailRule := analyzer.Rule{
		Name:        "email",
		Description: "Email addresses",
		Matcher:     pattern.EmailMatcher(),
		Settings: analyzer.RuleSettings{
			Strategy: analyzer.REDACT,
			Redact: &analyzer.RedactSettings{
				Placeholder: "<EMAIL_REDACTED>",
			},
		},
	}

	body := []byte("No sensitive data here")
	buffer := bytes.NewBuffer(nil)

	output, err := service.AnonymizeWithRules(context.Background(), []analyzer.Rule{emailRule}, body, buffer)
	require.NoError(t, err)
	assert.False(t, output.Details.HasFindings)
	assert.Equal(t, body, buffer.Bytes())
}
