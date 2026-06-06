package handler

import (
	"time"

	"github.com/uber-go/tally/v4"
)

// PrivacyMetrics tracks anonymization-related metrics
type PrivacyMetrics struct {
	scope tally.Scope
}

// NewPrivacyMetrics creates a new instance of PrivacyMetrics
func NewPrivacyMetrics(scope tally.Scope) PrivacyMetrics {
	return PrivacyMetrics{
		scope: scope.SubScope("privacy"),
	}
}

// ObserveRequestBodySize observes the size of the request body in bytes with a histogram
func (m PrivacyMetrics) ObserveRequestBodySize(bodySize int64) {
	m.scope.Histogram("request_body_size", tally.DefaultBuckets).RecordValue(float64(bodySize))
}

// ObserveAnonymizationDuration observes the duration of an anonymization operation in milliseconds
func (m PrivacyMetrics) ObserveAnonymizationDuration(duration time.Duration) {
	m.scope.Histogram("anonymization_duration", tally.DefaultBuckets).RecordDuration(duration)
}

// CountAnonymizedEntity increments the counter for anonymized entities with entity name as a tag
func (m PrivacyMetrics) CountAnonymizedEntity(entity string) {
	m.scope.Tagged(map[string]string{
		"entity": entity,
	}).Counter("anonymized_entity_count").Inc(1)
}
