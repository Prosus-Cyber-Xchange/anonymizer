package privacy_test

import (
	"testing"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/privacy"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleBuilder_Build_EmailWithRedaction(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	rules, err := builder.Build()

	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "EMAIL 0", rule.Name)
	assert.NotNil(t, rule.Matcher)
	assert.Equal(t, analyzer.REDACT, rule.Settings.Strategy)
	assert.NotNil(t, rule.Settings.Redact)
	assert.Equal(t, "<EMAIL_REDACTED>", rule.Settings.Redact.Placeholder)
	assert.False(t, rule.Disable)
}

func TestRuleBuilder_Build_CPFWithMasking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "CPF_NUMBER",
				Mask: &privacy.MaskSettings{
					Replacement: "*",
					MaxLength:   4,
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	rules, err := builder.Build()

	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Equal(t, "CPF_NUMBER 0", rule.Name)
	assert.NotNil(t, rule.Matcher)
	assert.Equal(t, analyzer.MASK, rule.Settings.Strategy)
	assert.NotNil(t, rule.Settings.Mask)
	assert.Equal(t, "*", rule.Settings.Mask.MaskingChar)
	assert.Equal(t, 4, rule.Settings.Mask.MaxSize)
}

func TestRuleBuilder_Build_MultipleEntities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
			},
			{
				Name: "CPF_NUMBER",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<CPF_REDACTED>",
				},
			},
			{
				Name: "IP_ADDRESS",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<IP_REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	rules, err := builder.Build()

	require.NoError(t, err)
	require.Len(t, rules, 3)

	assert.Equal(t, "EMAIL 0", rules[0].Name)
	assert.Equal(t, "CPF_NUMBER 1", rules[1].Name)
	assert.Equal(t, "IP_ADDRESS 2", rules[2].Name)
}

func TestRuleBuilder_Build_WithExceptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Exceptions: []privacy.ExceptionSettings{
					{
						Reason: "Public support email",
						Match: privacy.MatchSettings{
							Operator: pattern.MatchOperatorEqual,
							Pattern:  "[REDACTED_EMAIL]",
						},
					},
					{
						Reason: "Noreply emails",
						Match: privacy.MatchSettings{
							Operator: pattern.MatchOperatorStartsWith,
							Pattern:  "noreply@",
						},
					},
				},
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	rules, err := builder.Build()

	require.NoError(t, err)
	require.Len(t, rules, 1)

	rule := rules[0]
	assert.Len(t, rule.Exceptions, 2)
	assert.Equal(t, "Public support email", rule.Exceptions[0].Reason)
	assert.Equal(t, "Noreply emails", rule.Exceptions[1].Reason)
}

func TestRuleBuilder_Build_AllMatchOperators(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	operators := []struct {
		name     string
		operator string
		pattern  string
	}{
		{"equal", pattern.MatchOperatorEqual, "exact@example.com"},
		{"ignore case", pattern.MatchOperatorIgnoreCaseEqual, "CASE@EXAMPLE.COM"},
		{"starts with", pattern.MatchOperatorStartsWith, "noreply@"},
		{"ends with", pattern.MatchOperatorEndsWith, "@example.com"},
		{"regex", pattern.MatchOperatorRegex, `noreply\d+@example\.com`},
	}

	for _, tc := range operators {
		t.Run(tc.name, func(t *testing.T) {
			settings := privacy.PrivacySettings{
				Entities: []privacy.EntitySettings{
					{
						Name: "EMAIL",
						Exceptions: []privacy.ExceptionSettings{
							{
								Reason: "Test exception",
								Match: privacy.MatchSettings{
									Operator: tc.operator,
									Pattern:  tc.pattern,
								},
							},
						},
						Redaction: &privacy.RedactionSettings{
							Replacement: "<EMAIL_REDACTED>",
						},
					},
				},
			}

			builder := privacy.NewRuleBuilder(settings)
			rules, err := builder.Build()

			require.NoError(t, err)
			require.Len(t, rules, 1)
			require.Len(t, rules[0].Exceptions, 1)
			assert.NotNil(t, rules[0].Exceptions[0].Matcher)
		})
	}
}

func TestRuleBuilder_Build_UnsupportedEntity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "UNSUPPORTED_ENTITY",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	_, err := builder.Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported entity type")
}

func TestRuleBuilder_Build_InvalidMatchOperator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Exceptions: []privacy.ExceptionSettings{
					{
						Reason: "Test exception",
						Match: privacy.MatchSettings{
							Operator: "invalid_operator",
							Pattern:  "test",
						},
					},
				},
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	_, err := builder.Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported match operator")
}

func TestRuleBuilder_Build_InvalidRegexPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Exceptions: []privacy.ExceptionSettings{
					{
						Reason: "Test regex exception",
						Match: privacy.MatchSettings{
							Operator: pattern.MatchOperatorRegex,
							Pattern:  "[invalid(",
						},
					},
				},
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	_, err := builder.Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid regex pattern")
}

func TestRuleBuilder_Build_NoAnonymizationStrategy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				// No redaction or mask settings
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	_, err := builder.Build()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "neither redaction nor mask settings")
}

func TestRuleBuilder_Build_PreferRedactionOverMask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	settings := privacy.PrivacySettings{
		Entities: []privacy.EntitySettings{
			{
				Name: "EMAIL",
				Redaction: &privacy.RedactionSettings{
					Replacement: "<EMAIL_REDACTED>",
				},
				Mask: &privacy.MaskSettings{
					Replacement: "*",
					MaxLength:   4,
				},
			},
		},
	}

	builder := privacy.NewRuleBuilder(settings)
	rules, err := builder.Build()

	require.NoError(t, err)
	require.Len(t, rules, 1)

	// Should prefer redaction when both are defined
	rule := rules[0]
	assert.Equal(t, analyzer.REDACT, rule.Settings.Strategy)
	assert.NotNil(t, rule.Settings.Redact)
	assert.Nil(t, rule.Settings.Mask)
}

func TestRuleBuilder_Build_AllSupportedEntities(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	entities := []string{
		"EMAIL", "CPF_NUMBER", "CNPJ_NUMBER", "IP_ADDRESS",
		"CREDIT_CARD", "PHONE", "LINK", "SSN", "ADDRESS",
		"BANK_INFO", "UUID",
	}

	for _, entityName := range entities {
		t.Run(entityName, func(t *testing.T) {
			settings := privacy.PrivacySettings{
				Entities: []privacy.EntitySettings{
					{
						Name: entityName,
						Redaction: &privacy.RedactionSettings{
							Replacement: "<REDACTED>",
						},
					},
				},
			}

			builder := privacy.NewRuleBuilder(settings)
			rules, err := builder.Build()

			require.NoError(t, err, "Failed to build rule for entity %s", entityName)
			require.Len(t, rules, 1)
			assert.NotNil(t, rules[0].Matcher)
		})
	}
}
