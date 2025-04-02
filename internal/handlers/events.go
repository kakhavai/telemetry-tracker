package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	appmiddleware "github.com/kakhavain/telemetry-tracker/internal/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/storage"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// storer defines the interface for storing events.
type storer interface {
	StoreEvent(ctx context.Context, event storage.Event) error
	Close()
}

type eventTypeKey struct{}


// EventHandler handles incoming telemetry events.
type EventHandler struct {
	Store   storer
	Metrics *metrics.Registry
}

func NewEventHandler(store storer, metrics *metrics.Registry) *EventHandler {
	return &EventHandler{Store: store, Metrics: metrics}
}

// ServeHTTP handles POST requests to /events.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start a span for this handler.
	ctx, span := otel.Tracer("eventHandler").Start(r.Context(), "ServeHTTP")
	defer span.End()

	logger := appmiddleware.GetLoggerFromContext(ctx)
	if logger == nil {
		logger = slog.Default()
		logger.Warn("Logger not found in context for event handler")
	}

	event, status := parseEventRequest(r, logger)
	if status != http.StatusOK {
		http.Error(w, fmt.Sprintf("Request Error: %d", status), status)
		span.SetAttributes(attribute.Int("http.status_code", status))
		return
	}

	// Enrich logger with event type.
	logger = logger.With(slog.String("event_type", event.EventType))
	ctx = context.WithValue(ctx, eventTypeKey{}, event.EventType)

	// Record metric for events received.
	h.Metrics.EventsReceivedTotal.Add(ctx, 1,
		metric.WithAttributes(attribute.String("event_type", event.EventType)),
	)
	logger.Debug("Received event data")

	// Store the event.
	if err := h.Store.StoreEvent(ctx, event); err != nil {
		h.Metrics.DBErrorsTotal.Add(ctx, 1)
		logger.Error("Failed to store event", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to store event")
		return
	}
	
	h.Metrics.EventsStoredTotal.Add(ctx, 1)
	logger.Info("Event stored successfully")
	span.AddEvent("Event stored successfully", trace.WithAttributes(attribute.String("event_type", event.EventType)))
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "accepted"}`))
}

func parseEventRequest(r *http.Request, logger *slog.Logger) (storage.Event, int) {
	if r.Header.Get("Content-Type") != "application/json" {
		logger.Warn("Invalid content type", slog.String("content_type", r.Header.Get("Content-Type")))
		return storage.Event{}, http.StatusUnsupportedMediaType
	}

	var event storage.Event
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		logger.Warn("Failed to decode JSON body", slog.Any("error", err))
		return storage.Event{}, http.StatusBadRequest
	}

	if event.EventType == "" {
		logger.Warn("Missing 'event_type' field in request")
		return storage.Event{}, http.StatusBadRequest
	}

	return event, http.StatusOK
}
