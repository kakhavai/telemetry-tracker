package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kakhavain/telemetry-tracker/internal/observability"
)

func TestHealthHandler_Get(t *testing.T) {
	obs, _ := observability.InitObservability("noop")
	// Create a new instance of HealthHandler with observability.
	handler := NewHealthHandler(obs)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, status)
	}

	expectedBody := "OK"
	if rr.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, rr.Body.String())
	}

	expectedContentType := "text/plain; charset=utf-8"
	if rr.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected Content-Type %q, got %q", expectedContentType, rr.Header().Get("Content-Type"))
	}
}

func TestHealthHandler_InvalidMethod(t *testing.T) {
	obs, _ := observability.InitObservability("noop")
	handler := NewHealthHandler(obs)

	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}
