package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Get(t *testing.T) {
	// Create a new instance of HealthHandler.
	handler := &HealthHandler{}

	// Create a GET request.
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	// Use ResponseRecorder to record the response.
	rr := httptest.NewRecorder()

	// Serve the HTTP request.
	handler.ServeHTTP(rr, req)

	// Check that the status code is 200 OK.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status %v, got %v", http.StatusOK, status)
	}

	// Check that the response body is "OK".
	expectedBody := "OK"
	if rr.Body.String() != expectedBody {
		t.Errorf("expected body %q, got %q", expectedBody, rr.Body.String())
	}

	// Verify the Content-Type header.
	expectedContentType := "text/plain; charset=utf-8"
	if rr.Header().Get("Content-Type") != expectedContentType {
		t.Errorf("expected Content-Type %q, got %q", expectedContentType, rr.Header().Get("Content-Type"))
	}
}

func TestHealthHandler_InvalidMethod(t *testing.T) {
	handler := &HealthHandler{}

	// Create a request with a method other than GET.
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Expect a 405 Method Not Allowed status.
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("expected status %v, got %v", http.StatusMethodNotAllowed, status)
	}
}
