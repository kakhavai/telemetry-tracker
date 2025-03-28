package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	appmiddleware "github.com/kakhavain/telemetry-tracker/internal/middleware" // Use aliased import
	"github.com/kakhavain/telemetry-tracker/internal/storage"
)

// EventHandler handles incoming telemetry events.
type EventHandler struct {
	Store   storage.Storer
	Metrics *metrics.Registry
}

func NewEventHandler(store storage.Storer, metrics *metrics.Registry) *EventHandler {
	return &EventHandler{Store: store, Metrics: metrics}
}

// ServeHTTP handles POST requests to /events. Relies on middleware.
func (h *EventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := appmiddleware.GetLoggerFromContext(r.Context())
	if logger == nil {
		logger = slog.Default() // Fallback just in case
		logger.Warn("Logger not found in context for event handler")
	}

	// Basic check, although router should handle method enforcement
	if r.Method != http.MethodPost {
		logger.Warn("Method not allowed received in handler", slog.String("method", r.Method))
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Warn("Invalid content type", slog.String("content_type", r.Header.Get("Content-Type")))
		http.Error(w, "Content-Type header must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	var event storage.Event
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Prevent unexpected fields

	err := decoder.Decode(&event)
	if err != nil {
		logger.Warn("Failed to decode JSON body", slog.Any("error", err))
		http.Error(w, "Bad Request: Invalid JSON format", http.StatusBadRequest)
		return
	}
	// r.Body is closed automatically by the server

	// Basic validation
	if event.EventType == "" {
		logger.Warn("Missing 'event_type' field in request")
		http.Error(w, "Bad Request: Missing 'event_type' field", http.StatusBadRequest)
		return
	}

	// Add event type to logger context for subsequent logs related to this event
	logger = logger.With(slog.String("event_type", event.EventType))

	ctx := r.Context() // Use context from request, includes timeouts etc.

	h.Metrics.EventsReceivedTotal.WithLabelValues(event.EventType).Inc()
	logger.Debug("Received event data") // Use Debug for potentially noisy success path

	// Store the event
	err = h.Store.StoreEvent(ctx, event)
	if err != nil {
		h.Metrics.DBErrorsTotal.Inc()
		logger.Error("Failed to store event", slog.Any("error", err))
		// Return generic error to client
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	h.Metrics.EventsStoredTotal.Inc()
	logger.Info("Event stored successfully") // Use Info for significant success milestones

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status": "accepted"}`))
}