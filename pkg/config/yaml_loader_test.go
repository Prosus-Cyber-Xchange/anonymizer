package config_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Prosus-Cyber-Xchange/anonymizer/pkg/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLRulesLoader_Load_Success(t *testing.T) {
	basePath := filepath.Join("testdata")
	loader := config.NewYAMLRulesLoader(basePath)

	rules, err := loader.Load(context.Background(), "email_service")
	require.NoError(t, err)
	assert.Len(t, rules, 2) // EMAIL + CPF_NUMBER
}

func TestYAMLRulesLoader_Load_FileNotFound(t *testing.T) {
	basePath := filepath.Join("testdata")
	loader := config.NewYAMLRulesLoader(basePath)

	_, err := loader.Load(context.Background(), "nonexistent_service")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

func TestYAMLRulesLoader_Load_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "broken_service.yaml"), []byte("not: valid: yaml: ["), 0644)
	require.NoError(t, err)

	loader := config.NewYAMLRulesLoader(dir)
	_, err = loader.Load(context.Background(), "broken_service")
	assert.Error(t, err)
}
