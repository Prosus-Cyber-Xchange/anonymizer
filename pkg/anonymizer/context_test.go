package anonymizer_test

import (
	"context"
	"testing"

	"anonymizer-service-v2/pkg/anonymizer"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRules_RoundTrip(t *testing.T) {
	rules := []analyzer.Rule{
		{},
	}

	ctx := anonymizer.WithRules(context.Background(), rules)
	got, ok := anonymizer.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Len(t, got, 1)
}

func TestRulesFromContext_EmptyContext(t *testing.T) {
	_, ok := anonymizer.RulesFromContext(context.Background())
	assert.False(t, ok)
}

func TestWithRules_NilRules(t *testing.T) {
	ctx := anonymizer.WithRules(context.Background(), nil)
	got, ok := anonymizer.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Nil(t, got)
}
