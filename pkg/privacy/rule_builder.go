package privacy

import (
	"fmt"
	"strings"

	"github.com/Prosus-Cyber-Xchange/leakspok/analyzer"
	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
)

// GlobalExceptions defines exception matchers that are injected into every rule.
var GlobalExceptions = []ExceptionSettings{
	{
		Reason: "Git SSH address should not be redacted",
		Match: MatchSettings{
			Operator: pattern.MatchOperatorStartsWith,
			Pattern:  "git@",
		},
	},
	{
		Reason: "Go module version path should not be redacted",
		Match: MatchSettings{
			Operator: pattern.MatchOperatorRegex,
			Pattern:  `@v\d+\.\d+\.\d+$`,
		},
	},
}

// RuleBuilder converts privacy settings into leakspok analyzer rules
type RuleBuilder struct {
	settings PrivacySettings
}

// NewRuleBuilder creates a new rule builder with the provided settings
func NewRuleBuilder(settings PrivacySettings) *RuleBuilder {
	return &RuleBuilder{
		settings: settings,
	}
}

// Build converts privacy settings into a slice of leakspok analyzer rules
func (rb *RuleBuilder) Build() ([]analyzer.Rule, error) {
	rules := make([]analyzer.Rule, 0, len(rb.settings.Entities))

	for i, entityConfig := range rb.settings.Entities {
		rule, err := rb.buildRule(i, entityConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build rule for entity %s: %w", entityConfig.Name, err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

// buildRule converts a single entity configuration into a leakspok analyzer rule
func (rb *RuleBuilder) buildRule(ruleIndex int, entityConfig EntitySettings) (analyzer.Rule, error) {
	// Map entity name to leakspok matcher
	matcher, err := rb.getMatcherForEntity(entityConfig.Name)
	if err != nil {
		return analyzer.Rule{}, err
	}

	// Build exceptions: prepend global exceptions, then append per-entity exceptions
	allExceptions := make([]ExceptionSettings, 0, len(GlobalExceptions)+len(entityConfig.Exceptions))
	allExceptions = append(allExceptions, GlobalExceptions...)
	allExceptions = append(allExceptions, entityConfig.Exceptions...)

	exceptions := make([]analyzer.Exception, 0, len(allExceptions))
	for _, exceptionConfig := range allExceptions {
		exception, err := rb.buildException(exceptionConfig)
		if err != nil {
			return analyzer.Rule{}, fmt.Errorf("failed to build exception: %w", err)
		}
		exceptions = append(exceptions, exception)
	}

	// Build rule settings (anonymization strategy)
	ruleSettings, err := rb.buildRuleSettings(entityConfig)
	if err != nil {
		return analyzer.Rule{}, fmt.Errorf("failed to build rule settings: %w", err)
	}

	// Create the rule
	rule := analyzer.Rule{
		Name:        fmt.Sprintf("%s %d", entityConfig.Name, ruleIndex),
		Description: fmt.Sprintf("Anonymization rule for %s", entityConfig.Name),
		Matcher:     matcher,
		Exceptions:  exceptions,
		Settings:    ruleSettings,
		Disable:     false,
	}

	return rule, nil
}

// getMatcherForEntity maps entity names to leakspok pattern matchers
func (rb *RuleBuilder) getMatcherForEntity(entityName string) (analyzer.Matcher, error) {
	// Normalize entity name to uppercase for comparison
	normalizedName := strings.ToUpper(entityName)
	entityType := pattern.Entity(normalizedName)

	switch entityType {
	case pattern.EntityEmail:
		return pattern.EmailMatcher(), nil
	case pattern.EntityCPF:
		return pattern.CPFMatcher(), nil
	case pattern.EntityCNPJ:
		return pattern.CNPJMatcher(), nil
	case pattern.EntityIPAddress:
		return pattern.IPMatcher(), nil
	case pattern.EntityCreditCard:
		return pattern.CreditCardMatcher(), nil
	case pattern.EntityPhone:
		return pattern.PhoneMatcher(), nil
	case pattern.EntityLink:
		return pattern.LinkMatcher(), nil
	case pattern.EntitySSN:
		return pattern.SSNMatcher(), nil
	case pattern.EntityAddress:
		return pattern.AddressMatcher(), nil
	case pattern.EntityBankInfo:
		return pattern.BankInfoMatcher(), nil
	case pattern.EntityUUID:
		return pattern.UUIDMatcher(), nil
	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityName)
	}
}

// buildException converts an exception configuration into a leakspok exception
func (rb *RuleBuilder) buildException(exceptionConfig ExceptionSettings) (analyzer.Exception, error) {
	// Build pattern matcher based on operator
	matcher, err := rb.buildExceptionMatcher(exceptionConfig.Match)
	if err != nil {
		return analyzer.Exception{}, err
	}

	exception := analyzer.Exception{
		Reason:  exceptionConfig.Reason,
		Matcher: matcher,
	}

	return exception, nil
}

// buildExceptionMatcher creates a pattern matcher for exception matching
func (rb *RuleBuilder) buildExceptionMatcher(match MatchSettings) (analyzer.Matcher, error) {
	patternBytes := []byte(match.Pattern)

	var matchPattern pattern.Pattern

	switch strings.ToLower(match.Operator) {
	case pattern.MatchOperatorEqual:
		matchPattern = pattern.Equal(patternBytes)
	case pattern.MatchOperatorIgnoreCaseEqual:
		matchPattern = pattern.IgnoreCaseEqual(patternBytes)
	case pattern.MatchOperatorStartsWith:
		matchPattern = pattern.StartsWith(patternBytes)
	case pattern.MatchOperatorEndsWith:
		matchPattern = pattern.EndsWith(patternBytes)
	case pattern.MatchOperatorRegex:
		regexPattern, err := pattern.Regex(match.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", match.Pattern, err)
		}
		matchPattern = regexPattern
	default:
		return nil, fmt.Errorf("unsupported match operator: %s", match.Operator)
	}

	// Wrap the pattern in a PatternMatcher with a generic entity
	// Since this is for exceptions, we use a dummy entity name
	matcher := pattern.NewPatternMatcher("EXCEPTION", matchPattern)

	return matcher, nil
}

// buildRuleSettings converts entity settings into leakspok rule settings
func (rb *RuleBuilder) buildRuleSettings(entityConfig EntitySettings) (analyzer.RuleSettings, error) {
	// Determine strategy: prefer redaction if both are defined
	if entityConfig.Redaction != nil {
		return analyzer.RuleSettings{
			Strategy: analyzer.REDACT,
			Redact: &analyzer.RedactSettings{
				Placeholder: entityConfig.Redaction.Replacement,
			},
		}, nil
	}

	if entityConfig.Mask != nil {
		return analyzer.RuleSettings{
			Strategy: analyzer.MASK,
			Mask: &analyzer.MaskSettings{
				MaskingChar: entityConfig.Mask.Replacement,
				MaxSize:     entityConfig.Mask.MaxLength,
			},
		}, nil
	}

	// This should not happen if validation was done correctly
	return analyzer.RuleSettings{}, fmt.Errorf("entity %s has neither redaction nor mask settings", entityConfig.Name)
}
