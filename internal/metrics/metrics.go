package metrics

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Registry struct {
	Meter               metric.Meter
	EventsReceivedTotal metric.Int64Counter
	EventsStoredTotal   metric.Int64Counter
	DBErrorsTotal       metric.Int64Counter
	HTTPRequestTotal    metric.Int64Counter
	RequestDuration     metric.Float64Histogram
	ResponseSizeBytes   metric.Int64Histogram
}

func NewRegistry() (*Registry, error) {
	meter := otel.GetMeterProvider().Meter("telemetry-tracker")

	r := &Registry{Meter: meter}

	var err error

	if r.EventsReceivedTotal, err = meter.Int64Counter("telemetry_tracker.events_received_total"); err != nil {
		return nil, err
	}
	if r.EventsStoredTotal, err = meter.Int64Counter("telemetry_tracker.events_stored_total"); err != nil {
		return nil, err
	}
	if r.DBErrorsTotal, err = meter.Int64Counter("telemetry_tracker.database_errors_total"); err != nil {
		return nil, err
	}
	if r.HTTPRequestTotal, err = meter.Int64Counter("telemetry_tracker.http_requests_total"); err != nil {
		return nil, err
	}
	if r.RequestDuration, err = meter.Float64Histogram("telemetry_tracker.http_request_duration_seconds"); err != nil {
		return nil, err
	}
	if r.ResponseSizeBytes, err = meter.Int64Histogram("telemetry_tracker.http_response_size_bytes"); err != nil {
		return nil, err
	}

	return r, nil
}
