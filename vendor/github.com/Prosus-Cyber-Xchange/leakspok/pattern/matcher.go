package pattern

import "context"

// PatternMatcher associates an entity type with a detection pattern, forming
// a complete matcher that can be used in analyzer rules.
//
//nolint:revive // struct name is fine
type PatternMatcher struct {
	entity  Entity
	pattern Pattern
}

// NewPatternMatcher creates a PatternMatcher that associates the given entity
// type with the provided detection pattern.
func NewPatternMatcher(entity Entity, pattern Pattern) PatternMatcher {
	return PatternMatcher{
		entity:  entity,
		pattern: pattern,
	}
}

// Entity returns the entity type this matcher detects (e.g., "EMAIL", "CPF_NUMBER").
func (p PatternMatcher) Entity() Entity {
	return p.entity
}

// Match delegates to the underlying pattern to check whether the input matches.
func (p PatternMatcher) Match(ctx context.Context, input []byte) bool {
	return p.pattern.Match(ctx, input)
}

// PhoneMatcher returns a matcher for identifying international phone numbers
func PhoneMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityPhone,
		pattern: And(
			Not(
				Any(
					PatternFunc(MatchEmailV2),
					PatternFunc(MatchFilename),
					PatternFunc(MatchRepeatingNumber),
				),
			),
			Any(
				PatternFunc(MatchPhone),
			),
		),
	}
}

// LinkMatcher returns a matcher for identifying URLs and links that are not emails
func LinkMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityLink,
		pattern: And(
			Any(
				// matchlink,
				PatternFunc(MatchURL),
			),
			Not(
				PatternFunc(MatchEmailV2),
			),
		),
	}
}

// SSNMatcher returns a matcher for identifying US social security numbers
func SSNMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntitySSN,
		pattern: And(
			PatternFunc(MatchSSN),
			All(
				Not(PatternFunc(MatchPhoneV2)),
				Not(PatternFunc(MatchFilename)),
				Not(PatternFunc(MatchRepeatingNumber)),
			),
		),
	}
}

// EmailMatcher returns a matcher for identifying email addresses
func EmailMatcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityEmail,
		pattern: PatternFunc(MatchEmailV2),
	}
}

// IPMatcher returns a matcher for identifying IPv4 and IPv6 addresses
func IPMatcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityIPAddress,
		pattern: PatternFunc(MatchIP),
	}
}

// IPv4Matcher returns a matcher for identifying IP addresses
func IPv4Matcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityIPAddress,
		pattern: PatternFunc(MatchIPV4),
	}
}

// IPv6Matcher returns a matcher for identifying IPv6 addresses
func IPv6Matcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityIPAddress,
		pattern: PatternFunc(MatchIPV6),
	}
}

// CreditCardMatcher returns a matcher for identifying major credit card numbers
func CreditCardMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityCreditCard,
		pattern: And(
			Any(
				PatternFunc(MatchMasterCardCreditCard),
				PatternFunc(MatchVisaCreditCard),
			),
			All(
				Not(PatternFunc(MatchUUID)),
				Not(PatternFunc(MatchTestCreditCard)),
			),
		),
	}
}

// AddressMatcher returns a matcher for identifying street address, po boxes, and zip codes
func AddressMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityAddress,
		pattern: And(
			Any(
				PatternFunc(MatchStreetAddress),
				PatternFunc(MatchPOBox),
			),
			PatternFunc(MatchZipCode),
		),
	}
}

// BankInfoMatcher returns a matcher for identifying either IBANs or US Routing #s
func BankInfoMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityBankInfo,
		pattern: And(
			Any(
				PatternFunc(MatchIBAN),
			),
			Not(PatternFunc(MatchRepeatingNumber)),
		),
	}
}

// VINMatcher returns a matcher for identifying Vehicle Identification Numbers.
// VINs are exactly 17 alphanumeric characters excluding I, O, Q per ISO 3779.
func VINMatcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityVIN,
		pattern: PatternFunc(MatchVIN),
	}
}

// UUIDMatcher returns a matcher for identifying GUIDs, UUIDs, v3, v4, and v5
func UUIDMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityUUID,
		pattern: And(
			Any(
				PatternFunc(Matchguid),
				PatternFunc(MatchUUID),
				PatternFunc(MatchUUIDV3),
				PatternFunc(MatchUUIDV4),
				PatternFunc(MatchUUIDV5),
			),
			Not(
				PatternFunc(MatchFilename),
			),
		),
	}
}

// HaltLangDetect is a special matcher for preventing language detection from running
func HaltLangDetect() Pattern {
	return Any(
		UUIDMatcher().pattern,
		LinkMatcher().pattern,
		EmailMatcher().pattern,
		CreditCardMatcher().pattern,
	)
}

// CPFMatcher generates a matcher for identifying Brazilian CPFs
func CPFMatcher() PatternMatcher {
	return PatternMatcher{
		entity:  EntityCPF,
		pattern: PatternFunc(MatchCPF),
	}
}

// CNPJMatcher generates a matcher for identifying Brazilian CNPJs
func CNPJMatcher() PatternMatcher {
	return PatternMatcher{
		entity: EntityCNPJ,
		pattern: Any(
			PatternFunc(MatchCNPJ),
			PatternFunc(MatchCNPJV2),
		),
	}
}

// BrazilianPIIMatcher generates a matcher for identifying Brazilian identification numbers
func BrazilianPIIMatcher() PatternMatcher {
	return PatternMatcher{
		entity: "BRAZILIAN_PII",
		pattern: Any(
			PatternFunc(MatchCPF),
			PatternFunc(MatchCNPJ),
			PatternFunc(MatchCNPJV2),
		),
	}
}
