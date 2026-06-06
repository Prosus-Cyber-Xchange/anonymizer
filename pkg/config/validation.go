package config

import (
	"fmt"
	"strings"

	"anonymizer-service-v2/pkg/privacy"

	"github.com/Prosus-Cyber-Xchange/leakspok/pattern"
	"github.com/asaskevich/govalidator"
)

// ValidatePrivacyConfig validates the privacy configuration using structured validation rules
func ValidatePrivacyConfig(settings privacy.PrivacySettings) error {
	// Validate entities field is not empty or null
	if len(settings.Entities) == 0 {
		return fmt.Errorf("entities field cannot be empty or null")
	}

	// Validate each entity
	for i, entity := range settings.Entities {
		if err := validateEntitySettings(entity, i); err != nil {
			return err
		}
	}

	return nil
}

func validateEntitySettings(entity privacy.EntitySettings, index int) error {
	// EntitySetting must have a name
	if !govalidator.IsNotNull(entity.Name) || entity.Name == "" {
		return fmt.Errorf("entity at index %d must have a non-empty name", index)
	}

	// EntitySettings must have Redaction OR Mask defined. Both empty should be invalid.
	if entity.Redaction == nil && entity.Mask == nil {
		return fmt.Errorf("entity %q must define either redaction or mask settings", entity.Name)
	}

	// Validate redaction settings if present
	if entity.Redaction != nil {
		if err := validateRedactionSettings(entity.Redaction, entity.Name); err != nil {
			return err
		}
	}

	// Validate mask settings if present
	if entity.Mask != nil {
		if err := validateMaskSettings(entity.Mask, entity.Name); err != nil {
			return err
		}
	}

	// Validate exceptions
	for j, exception := range entity.Exceptions {
		if err := validateExceptionSettings(exception, entity.Name, j); err != nil {
			return err
		}
	}

	return nil
}

func validateRedactionSettings(redaction *privacy.RedactionSettings, entityName string) error {
	// RedactionSettings must have a Replacement field
	if govalidator.IsNull(redaction.Replacement) {
		return fmt.Errorf("entity %q redaction must have a non-empty replacement field", entityName)
	}

	return nil
}

func validateMaskSettings(mask *privacy.MaskSettings, entityName string) error {
	// MaskSettings must have both Replacement and MaxLength
	if govalidator.IsNull(mask.Replacement) {
		return fmt.Errorf("entity %q mask must have a non-empty replacement field", entityName)
	}

	if mask.MaxLength <= 0 {
		return fmt.Errorf("entity %q mask must have a valid maxLength greater than 0", entityName)
	}

	return nil
}

func validateExceptionSettings(exception privacy.ExceptionSettings, entityName string, index int) error {
	// Exception Match.Operator must be one of the following: equal, ignoreCaseEqual, startsWith, endsWith
	operator := strings.ToLower(exception.Match.Operator)
	isOperatorValid := govalidator.IsIn(operator,
		pattern.MatchOperatorEqual,
		pattern.MatchOperatorIgnoreCaseEqual,
		pattern.MatchOperatorStartsWith,
		pattern.MatchOperatorEndsWith)

	if !isOperatorValid {
		return fmt.Errorf(
			"entity %q exception at index %d has invalid operator %q (valid: %s, %s, %s, %s)",
			entityName, index, exception.Match.Operator,
			pattern.MatchOperatorEqual,
			pattern.MatchOperatorIgnoreCaseEqual,
			pattern.MatchOperatorStartsWith,
			pattern.MatchOperatorEndsWith,
		)
	}

	// Exception Match.Pattern must not be empty
	if govalidator.IsNull(exception.Match.Pattern) {
		return fmt.Errorf("entity %q exception at index %d has empty pattern", entityName, index)
	}

	return nil
}
