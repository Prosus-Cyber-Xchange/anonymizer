package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"anonymizer-service-v2/pkg/privacy"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"gopkg.in/yaml.v3"
)

// YAMLRulesLoader loads privacy rules from YAML files on disk.
// Each service has a file named <service_name>.yaml in the base path.
type YAMLRulesLoader struct {
	basePath string
}

// NewYAMLRulesLoader creates a loader that reads YAML files from basePath.
func NewYAMLRulesLoader(basePath string) *YAMLRulesLoader {
	return &YAMLRulesLoader{basePath: basePath}
}

// Load reads the YAML file for the given service and builds analyzer rules.
func (l *YAMLRulesLoader) Load(_ context.Context, serviceName string) ([]analyzer.Rule, error) {
	filePath := filepath.Join(l.basePath, serviceName+".yaml")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read rules file %s: %w", filePath, err)
	}

	var settings privacy.PrivacySettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse YAML rules for %s: %w", serviceName, err)
	}

	rules, err := privacy.NewRuleBuilder(settings).Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build rules for %s: %w", serviceName, err)
	}

	return rules, nil
}
