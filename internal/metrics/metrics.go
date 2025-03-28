// internal/metrics/metrics.go

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	// Import the new 'collectors' package for Go runtime metrics
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// Registry holds the Prometheus metrics collectors and the registry itself.
type Registry struct {
	Reg                 *prometheus.Registry // The custom registry
	EventsReceivedTotal *prometheus.CounterVec
	EventsStoredTotal   prometheus.Counter
	DBErrorsTotal       prometheus.Counter
	// HTTP Metrics used by middleware
	HTTPRequestTotal    *prometheus.CounterVec   // For total requests
	RequestDuration     *prometheus.HistogramVec // For duration
	ResponseSize        *prometheus.HistogramVec // For response size
}

// NewRegistry creates and registers the metrics collectors in a new registry.
func NewRegistry() *Registry {
	// Create a non-global registry to avoid potential conflicts
	reg := prometheus.NewRegistry()

	// Initialize the Registry struct where metrics will be stored
	metrics := &Registry{
		Reg: reg, // Store the registry itself

		// --- Application Specific Metrics ---
		EventsReceivedTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "telemetry_tracker", // Use a namespace for clarity
				Name:      "events_received_total",
				Help:      "Total number of events received via the HTTP endpoint.",
			},
			[]string{"event_type"}, // Label by event type
		),
		EventsStoredTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "telemetry_tracker",
				Name:      "events_stored_total",
				Help:      "Total number of events successfully stored in the database.",
			},
		),
		DBErrorsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "telemetry_tracker",
				Name:      "database_errors_total",
				Help:      "Total number of database errors encountered during storage.",
			},
		),

		// --- HTTP Middleware Metrics ---
		HTTPRequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "telemetry_tracker",
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests processed.",
			},
			[]string{"code", "method", "path"}, // Standard labels populated by promhttp middleware
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "telemetry_tracker",
				Name:      "http_request_duration_seconds", // Standard name
				Help:      "Histogram of HTTP request durations in seconds.",
				Buckets:   prometheus.DefBuckets,        // Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10 seconds
			},
			[]string{"code", "method", "path"}, // Match labels with counter and size
		),
		ResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "telemetry_tracker",
				Name:      "http_response_size_bytes", // Standard name
				Help:      "Histogram of HTTP response sizes in bytes.",
				// Example buckets: 100B, 1KB, 10KB, 100KB, 1MB, 10MB
				Buckets:   prometheus.ExponentialBuckets(100, 10, 6),
			},
			[]string{"code", "method", "path"}, // Match labels with counter and duration
		),
	}

	// --- Register Custom Metrics ---
	reg.MustRegister(metrics.EventsReceivedTotal)
	reg.MustRegister(metrics.EventsStoredTotal)
	reg.MustRegister(metrics.DBErrorsTotal)
	reg.MustRegister(metrics.HTTPRequestTotal)
	reg.MustRegister(metrics.RequestDuration)
	reg.MustRegister(metrics.ResponseSize)

	// --- Register Standard Collectors ---

	// Register standard Go runtime metrics using the NEW collectors package
	// This replaces the deprecated prometheus.NewGoCollector
	reg.MustRegister(collectors.NewGoCollector(
		collectors.WithGoCollectorRuntimeMetrics(collectors.GoRuntimeMetricsRule{
			// CORRECTED THIS LINE: It's GoRuntimeMetricsRuleAlways
			Matcher: collectors.MetricsAll.Matcher,
		}),
	))

	// Register process metrics
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{
		Namespace: "telemetry_tracker",
	}))
	//NewProcessCollector also moved to the collectors package. Updated this line too.

	return metrics // Return the registry struct containing all metrics and the *prometheus.Registry
}