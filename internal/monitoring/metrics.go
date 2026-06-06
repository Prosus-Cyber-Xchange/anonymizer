package monitoring

import (
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/uber-go/tally/v4"
	tallyprom "github.com/uber-go/tally/v4/prometheus"
)

func NewMetricRegistry() *prometheus.Registry {
	return prometheus.NewRegistry()
}

func NewPrometheusScope(registry *prometheus.Registry, reportInterval time.Duration) (tally.Scope, io.Closer, http.Handler) {
	defaultHistogramBuckets := []float64{
		0.001, // 1ms
		0.005, // 5ms
		0.01,  // 10ms
		0.025, // 25ms
		0.05,  // 50ms
		0.075, // 75ms
		0.1,   // 100ms
		0.15,  // 150ms
		0.2,   // 200ms
		0.5,   // 500ms
		0.75,  // 750ms
		1,     // 1s
		1.5,   // 1.5s
		2,     // 2s
		3,     // 3s
		4,     // 4s
		6,     // 6s
		8,     // 8s
		10,    // 10s
		20,    // 20s
	}

	reporter := tallyprom.NewReporter(tallyprom.Options{
		DefaultHistogramBuckets: defaultHistogramBuckets,
	})
	tags := map[string]string{}

	scope, closer := tally.NewRootScope(tally.ScopeOptions{
		CachedReporter: reporter,
		Tags:           tags,
		Prefix:         "yggdrasil_authorizer",
		Separator:      tallyprom.DefaultSeparator,
		// Below is necessary because of unsolved issue https://github.com/uber-go/tally/issues/256
		SanitizeOptions: &tallyprom.DefaultSanitizerOpts,
	}, reportInterval)

	return scope, closer, reporter.HTTPHandler()
}
