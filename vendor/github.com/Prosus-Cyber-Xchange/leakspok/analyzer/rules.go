package analyzer

import (
	"context"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

//go:generate go run -mod=mod go.uber.org/mock/mockgen -destination=mocks/mock_rules.go -package=analyzermock -source=$GOFILE

// Matcher defines the interface for detecting patterns in byte slices.
// Each implementation identifies a specific entity type (e.g., email, CPF)
// and checks whether the input matches that pattern.
type Matcher interface {
	Entity() pattern.Entity
	Match(ctx context.Context, input []byte) bool // we could return a more complex type, maybe with error, to allow context cancellation propagation
}

// RedactSettings configures how a matched token is redacted.
type RedactSettings struct {
	Placeholder string `json:"placeholder,omitempty"`
}

// MaskSettings configures how a matched token is masked.
// The first MaxSize characters of the token are replaced with MaskingChar.
type MaskSettings struct {
	MaskingChar string `json:"maskingChar,omitempty"`
	MaxSize     int    `json:"maxSize,omitempty"`
}

// AnonymizeStrategy defines the settings for anonymizing a finding
type AnonymizeStrategy int

const (
	// REDACT is the settings for redacting a finding
	REDACT AnonymizeStrategy = iota
	// MASK is the settings for masking a finding
	MASK
)

// Exception allows whitelisting specific patterns from a detection rule.
// When a token matches an exception's matcher, the rule is skipped.
type Exception struct {
	Reason  string  `json:"reason,omitempty"`
	Matcher Matcher `json:"-"`
}

// RuleSettings configures how a detection rule handles matches.
// Strategy selects between REDACT and MASK; the corresponding settings
// field provides the configuration.
type RuleSettings struct {
	Strategy AnonymizeStrategy `json:"settings"`
	Redact   *RedactSettings   `json:"redact,omitempty"`
	Mask     *MaskSettings     `json:"mask,omitempty"`
}

// Rule defines a detection rule that identifies and anonymizes a specific
// type of sensitive data. Each rule has a matcher for detection, optional
// exceptions for whitelisting known-safe patterns, and settings that
// control how matches are anonymized.
type Rule struct {
	Disable     bool         `json:"disable"`
	Name        string       `json:"name,omitempty"`
	Description string       `json:"description,omitempty"`
	Matcher     Matcher      `json:"-"`
	Exceptions  []Exception  `json:"exceptions,omitempty"`
	Settings    RuleSettings `json:"anonymize,omitempty"`
}
