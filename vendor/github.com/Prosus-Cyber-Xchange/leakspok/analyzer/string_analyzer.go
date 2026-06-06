package analyzer

import (
	"bytes"
	"context"
	"log/slog"
)

// StringAnalyzer wraps ByteAnalyzer to work with strings instead of byte slices
type StringAnalyzer struct {
	byteAnalyzer ByteAnalyzer
}

// NewStringAnalyzer creates a new StringAnalyzer with the provided logger and RuleRunner
func NewStringAnalyzer(logger *slog.Logger, runner RuleRunner) StringAnalyzer {
	return StringAnalyzer{
		byteAnalyzer: NewByteAnalyzer(logger, runner),
	}
}

// Anonymize anonymizes matches within a string and returns the anonymized string and anonymization details
func (sa *StringAnalyzer) Anonymize(ctx context.Context, rules []Rule, input string) (string, AnonymizationDetails) {
	// Convert string to []byte
	inputBytes := []byte(input)

	// Create a buffer to collect the anonymized output
	// Let the client code decide if buffering is needed
	buf := &bytes.Buffer{}

	// Delegate to ByteAnalyzer
	details := sa.byteAnalyzer.Anonymize(ctx, rules, buf, inputBytes)

	// Convert result back to string
	return buf.String(), details
}

// Stop releases the underlying ByteAnalyzer's resources. It is safe to call
// concurrently with Anonymize.
func (sa *StringAnalyzer) Stop() {
	sa.byteAnalyzer.Stop()
}
