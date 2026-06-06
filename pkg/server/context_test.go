package server_test

import (
	"context"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/server"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithRules_RoundTrip(t *testing.T) {
	rules := []analyzer.Rule{
		{},
	}

	ctx := server.WithRules(context.Background(), rules)
	got, ok := server.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Len(t, got, 1)
}

func TestRulesFromContext_EmptyContext(t *testing.T) {
	_, ok := server.RulesFromContext(context.Background())
	assert.False(t, ok)
}

func TestWithRules_NilRules(t *testing.T) {
	ctx := server.WithRules(context.Background(), nil)
	got, ok := server.RulesFromContext(ctx)

	require.True(t, ok)
	assert.Nil(t, got)
}
