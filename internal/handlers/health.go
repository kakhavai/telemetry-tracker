package handlers

import (
	"net/http"

	"go.opentelemetry.io/otel"
)

// HealthHandler provides a simple health check endpoint.
type HealthHandler struct{}

// ServeHTTP handles GET requests to /healthz.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Start a span for health check.
	_, span := otel.Tracer("healthHandler").Start(r.Context(), "HealthCheck")
	defer span.End()

	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
