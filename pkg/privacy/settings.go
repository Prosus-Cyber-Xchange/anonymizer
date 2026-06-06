package privacy

import "github.com/Prosus-Cyber-Xchange/leakspok/analyzer"

// PrivacySettings represents the root privacy configuration loaded from YAML
type PrivacySettings struct {
	Entities []EntitySettings `yaml:"entities"`
}

// EntitySettings represents a single entity configuration
type EntitySettings struct {
	Name       string              `yaml:"name"`
	Exceptions []ExceptionSettings `yaml:"exceptions,omitempty"`
	Redaction  *RedactionSettings  `yaml:"redaction,omitempty"`
	Mask       *MaskSettings       `yaml:"mask,omitempty"`
}

// ExceptionSettings represents an exception rule for an entity
type ExceptionSettings struct {
	Reason string        `yaml:"reason"`
	Match  MatchSettings `yaml:"match"`
}

// MatchSettings represents the matching condition for an exception
type MatchSettings struct {
	Operator string `yaml:"operator"` // equal, ignoreCaseEqual, startsWith, endsWith
	Pattern  string `yaml:"pattern"`
}

// RedactionSettings defines how to redact an entity
type RedactionSettings struct {
	Replacement string `yaml:"replacement"`
}

// MaskSettings defines how to mask an entity
type MaskSettings struct {
	Replacement string `yaml:"replacement"` // Character used for masking
	MaxLength   int    `yaml:"maxLength"`   // Maximum length of masked string
}

// AnonymizeOutput represents the result of an anonymization operation
type AnonymizeOutput struct {
	Details analyzer.AnonymizationDetails
}
