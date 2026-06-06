package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAnonymizeHeaders_ValidEntities(t *testing.T) {
	headers := map[string]string{
		HeaderEntities: "EMAIL,CPF_NUMBER",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Len(t, settings.Entities, 2)
	assert.Equal(t, "EMAIL", settings.Entities[0].Name)
	assert.Equal(t, "CPF_NUMBER", settings.Entities[1].Name)
	// Default strategy: redact with <REDACTED>
	assert.NotNil(t, settings.Entities[0].Redaction)
	assert.Equal(t, "<REDACTED>", settings.Entities[0].Redaction.Replacement)
}

func TestParseAnonymizeHeaders_CustomPlaceholder(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:    "EMAIL",
		HeaderPlaceholder: "[REMOVED]",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Equal(t, "[REMOVED]", settings.Entities[0].Redaction.Replacement)
}

func TestParseAnonymizeHeaders_MaskStrategy(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:   "EMAIL",
		HeaderStrategy:   "mask",
		HeaderMaskChar:   "#",
		HeaderMaskLength: "6",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Nil(t, settings.Entities[0].Redaction)
	assert.NotNil(t, settings.Entities[0].Mask)
	assert.Equal(t, "#", settings.Entities[0].Mask.Replacement)
	assert.Equal(t, 6, settings.Entities[0].Mask.MaxLength)
}

func TestParseAnonymizeHeaders_MissingEntities(t *testing.T) {
	headers := map[string]string{}

	_, err := parseAnonymizeHeaders(headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "X-Anonymize-Entities")
}

func TestParseAnonymizeHeaders_InvalidMaskLength(t *testing.T) {
	headers := map[string]string{
		HeaderEntities:   "EMAIL",
		HeaderStrategy:   "mask",
		HeaderMaskLength: "abc",
	}

	_, err := parseAnonymizeHeaders(headers)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mask-length")
}

func TestParseAnonymizeHeaders_TrimWhitespace(t *testing.T) {
	headers := map[string]string{
		HeaderEntities: " EMAIL , CPF_NUMBER ",
	}

	settings, err := parseAnonymizeHeaders(headers)
	require.NoError(t, err)
	assert.Equal(t, "EMAIL", settings.Entities[0].Name)
	assert.Equal(t, "CPF_NUMBER", settings.Entities[1].Name)
}
