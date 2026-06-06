package privacy

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"

	"github.com/Prosus-Cyber-Xchange/anonymizer/internal/monitoring"
)

// Service provides anonymization functionality using leakspok
type Service struct {
	logger       *slog.Logger
	byteAnalyzer analyzer.ByteAnalyzer
}

// NewService creates a new privacy service
func NewService(byteAnalyzer analyzer.ByteAnalyzer, logger *slog.Logger) *Service {
	return &Service{
		logger:       logger,
		byteAnalyzer: byteAnalyzer,
	}
}

// AnonymizeWithRules performs anonymization with pre-loaded rules and writes output to the provided writer
func (s *Service) AnonymizeWithRules(ctx context.Context, rules []analyzer.Rule, input []byte, output io.Writer) (AnonymizeOutput, error) {
	// Create span for anonymization
	span, ctx := monitoring.StartSpan(ctx, "anonymize")
	monitoring.SetTag(span, "anonymization.input_size", len(input))
	monitoring.SetTag(span, "anonymization.rules_count", len(rules))
	defer span.Finish()

	if len(input) == 0 {
		// If body is empty, skip anonymization
		monitoring.SetTag(span, "anonymization.skipped_reason", "empty_input")
		return AnonymizeOutput{}, nil
	}

	if len(rules) == 0 {
		// No rules to apply, copy input directly to output
		monitoring.SetTag(span, "anonymization.skipped_reason", "no_rules")
		if _, err := output.Write(input); err != nil {
			return AnonymizeOutput{}, fmt.Errorf("failed to write to output: %w", err)
		}
		return AnonymizeOutput{}, nil
	}

	details := s.byteAnalyzer.Anonymize(ctx, rules, output, input)

	monitoring.SetTag(span, "anonymization.has_findings", details.HasFindings)

	return AnonymizeOutput{
		Details: details,
	}, nil
}
