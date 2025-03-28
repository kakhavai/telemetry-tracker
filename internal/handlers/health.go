package handlers

import (
	"net/http"
)

// HealthHandler provides a simple health check endpoint.
type HealthHandler struct{}

// ServeHTTP handles GET requests to /healthz
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Basic check, although router should handle method enforcement
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}