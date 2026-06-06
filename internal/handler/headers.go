package handler

import (
	"fmt"
	"strconv"
	"strings"

	"anonymizer-service-v2/pkg/privacy"
)

// Header constants for content negotiation.
const (
	HeaderEntities    = "X-Anonymize-Entities"
	HeaderStrategy    = "X-Anonymize-Strategy"
	HeaderPlaceholder = "X-Anonymize-Placeholder"
	HeaderMaskChar    = "X-Anonymize-Mask-Char"
	HeaderMaskLength  = "X-Anonymize-Mask-Length"

	HeaderDetectedEntities   = "X-Anonymize-Detected-Entities"
	HeaderAnonymizedEntities = "X-Anonymize-Anonymized-Entities"
)

// parseAnonymizeHeaders builds PrivacySettings from X-Anonymize-* request headers.
// headers is a map of header-name → value (caller extracts from http.Request).
func parseAnonymizeHeaders(headers map[string]string) (privacy.PrivacySettings, error) {
	entitiesRaw, ok := headers[HeaderEntities]
	if !ok || strings.TrimSpace(entitiesRaw) == "" {
		return privacy.PrivacySettings{}, fmt.Errorf("%s header is required when no rules are pre-configured", HeaderEntities)
	}

	strategy := strings.ToLower(strings.TrimSpace(headers[HeaderStrategy]))
	if strategy == "" {
		strategy = "redact"
	}

	entityNames := strings.Split(entitiesRaw, ",")
	entities := make([]privacy.EntitySettings, 0, len(entityNames))

	for _, name := range entityNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		entity := privacy.EntitySettings{Name: strings.ToUpper(name)}

		switch strategy {
		case "redact":
			placeholder := headers[HeaderPlaceholder]
			if placeholder == "" {
				placeholder = "<REDACTED>"
			}
			entity.Redaction = &privacy.RedactionSettings{Replacement: placeholder}

		case "mask":
			maskChar := headers[HeaderMaskChar]
			if maskChar == "" {
				maskChar = "*"
			}
			maskLength := 4
			if ml, exists := headers[HeaderMaskLength]; exists && ml != "" {
				parsed, err := strconv.Atoi(ml)
				if err != nil {
					return privacy.PrivacySettings{}, fmt.Errorf("invalid mask-length value %q: must be an integer", ml)
				}
				maskLength = parsed
			}
			entity.Mask = &privacy.MaskSettings{Replacement: maskChar, MaxLength: maskLength}

		default:
			return privacy.PrivacySettings{}, fmt.Errorf("invalid %s value %q: must be 'redact' or 'mask'", HeaderStrategy, strategy)
		}

		entities = append(entities, entity)
	}

	return privacy.PrivacySettings{Entities: entities}, nil
}
