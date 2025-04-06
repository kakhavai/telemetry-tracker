package handlers

import (
	"net/http"

	"github.com/kakhavain/telemetry-tracker/internal/observability"
)

// HealthHandler provides a simple health check endpoint.
type HealthHandler struct {
	Obs observability.Provider
}

// NewHealthHandler constructs a HealthHandler with observability.
func NewHealthHandler(obs observability.Provider) *HealthHandler {
	return &HealthHandler{Obs: obs}
}

// ServeHTTP handles GET requests to /healthz.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start a span for health check.
	_, span := h.Obs.Tracer().Start(r.Context(), "HealthCheck")
	defer span.End()

	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
