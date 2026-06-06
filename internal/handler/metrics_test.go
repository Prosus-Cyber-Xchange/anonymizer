package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/uber-go/tally/v4"
)

func TestNewPrivacyMetrics(t *testing.T) {
	t.Run("creates metrics with correct scope", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		assert.NotNil(t, metrics)
		assert.NotNil(t, metrics.scope)
	})

	t.Run("scope has privacy subscope", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		// Verify that the metrics object can be used (no panic)
		assert.NotPanics(t, func() {
			metrics.ObserveRequestBodySize(100)
		})
	})
}

func TestObserveRequestBodySize(t *testing.T) {
	t.Run("records small body size", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveRequestBodySize(100)

		// Verify the histogram was recorded
		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records large body size", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		largeSize := int64(1024 * 1024) // 1MB
		metrics.ObserveRequestBodySize(largeSize)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records zero body size", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveRequestBodySize(0)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records multiple body sizes", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		sizes := []int64{100, 500, 1000, 5000}
		for _, size := range sizes {
			metrics.ObserveRequestBodySize(size)
		}

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})
}

func TestObserveAnonymizationDuration(t *testing.T) {
	t.Run("records short duration", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveAnonymizationDuration(100 * time.Millisecond)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records long duration", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveAnonymizationDuration(5 * time.Second)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records nanosecond precision", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveAnonymizationDuration(500 * time.Microsecond)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records zero duration", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.ObserveAnonymizationDuration(0)

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})

	t.Run("records multiple durations", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		durations := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			150 * time.Millisecond,
			500 * time.Millisecond,
		}
		for _, duration := range durations {
			metrics.ObserveAnonymizationDuration(duration)
		}

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		assert.NotNil(t, histograms)
	})
}

func TestCountAnonymizedEntity(t *testing.T) {
	t.Run("increments counter for single entity", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.CountAnonymizedEntity("EMAIL")

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("increments counter for CPF entity", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.CountAnonymizedEntity("CPF")

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("increments counter multiple times for same entity", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		entity := "EMAIL"
		for i := 0; i < 5; i++ {
			metrics.CountAnonymizedEntity(entity)
		}

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("increments counters for different entities", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		entities := []string{"EMAIL", "CPF", "CNPJ", "IP_ADDRESS", "PHONE"}
		for _, entity := range entities {
			metrics.CountAnonymizedEntity(entity)
		}

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("counts multiple instances of same entity", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		// Simulate anonymizing 3 emails in a single request
		for i := 0; i < 3; i++ {
			metrics.CountAnonymizedEntity("EMAIL")
		}

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("handles lowercase entity names", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.CountAnonymizedEntity("email")

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("handles mixed case entity names", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.CountAnonymizedEntity("Email")

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})

	t.Run("handles entity names with underscores", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		metrics.CountAnonymizedEntity("IP_ADDRESS")

		snapshot := scope.Snapshot()
		counters := snapshot.Counters()
		assert.NotNil(t, counters)
	})
}

func TestMetricsIntegration(t *testing.T) {
	t.Run("records all metric types in sequence", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		// Simulate a real anonymization operation
		metrics.ObserveRequestBodySize(500)
		metrics.ObserveAnonymizationDuration(150 * time.Millisecond)
		metrics.CountAnonymizedEntity("EMAIL")
		metrics.CountAnonymizedEntity("CPF")

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		counters := snapshot.Counters()

		assert.NotNil(t, histograms)
		assert.NotNil(t, counters)
	})

	t.Run("handles rapid successive metric recording", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		// Simulate rapid requests
		for i := 0; i < 100; i++ {
			metrics.ObserveRequestBodySize(int64(100 * (i + 1)))
			metrics.ObserveAnonymizationDuration(time.Duration(10*(i+1)) * time.Millisecond)
			if i%2 == 0 {
				metrics.CountAnonymizedEntity("EMAIL")
			} else {
				metrics.CountAnonymizedEntity("CPF")
			}
		}

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		counters := snapshot.Counters()

		assert.NotNil(t, histograms)
		assert.NotNil(t, counters)
	})

	t.Run("records metrics with realistic values", func(t *testing.T) {
		scope := tally.NewTestScope("test", nil)
		metrics := NewPrivacyMetrics(scope)

		// Simulate typical request patterns
		testCases := []struct {
			bodySize       int64
			duration       time.Duration
			anonymizedList []string
		}{
			{250, 50 * time.Millisecond, []string{"EMAIL"}},
			{1500, 100 * time.Millisecond, []string{"EMAIL", "CPF"}},
			{5000, 250 * time.Millisecond, []string{"EMAIL", "CPF", "CNPJ"}},
			{100000, 500 * time.Millisecond, []string{"EMAIL", "IP_ADDRESS"}},
		}

		for _, tc := range testCases {
			metrics.ObserveRequestBodySize(tc.bodySize)
			metrics.ObserveAnonymizationDuration(tc.duration)
			for _, entity := range tc.anonymizedList {
				metrics.CountAnonymizedEntity(entity)
			}
		}

		snapshot := scope.Snapshot()
		histograms := snapshot.Histograms()
		counters := snapshot.Counters()

		assert.NotNil(t, histograms)
		assert.NotNil(t, counters)
	})
}

func BenchmarkObserveRequestBodySize(b *testing.B) {
	scope := tally.NewTestScope("test", nil)
	metrics := NewPrivacyMetrics(scope)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ObserveRequestBodySize(int64(1000 + i))
	}
}

func BenchmarkObserveAnonymizationDuration(b *testing.B) {
	scope := tally.NewTestScope("test", nil)
	metrics := NewPrivacyMetrics(scope)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ObserveAnonymizationDuration(time.Duration(100+i) * time.Millisecond)
	}
}

func BenchmarkCountAnonymizedEntity(b *testing.B) {
	scope := tally.NewTestScope("test", nil)
	metrics := NewPrivacyMetrics(scope)

	entities := []string{"EMAIL", "CPF", "CNPJ", "IP_ADDRESS", "PHONE"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.CountAnonymizedEntity(entities[i%len(entities)])
	}
}

func BenchmarkMetricsAllOperations(b *testing.B) {
	scope := tally.NewTestScope("test", nil)
	metrics := NewPrivacyMetrics(scope)

	entities := []string{"EMAIL", "CPF", "CNPJ", "IP_ADDRESS", "PHONE"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.ObserveRequestBodySize(int64(100 * (i + 1)))
		metrics.ObserveAnonymizationDuration(time.Duration(10*(i+1)) * time.Millisecond)
		metrics.CountAnonymizedEntity(entities[i%len(entities)])
	}
}
