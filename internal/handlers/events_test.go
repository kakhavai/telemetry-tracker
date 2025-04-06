package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/kakhavain/telemetry-tracker/internal/handlers"
	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	appmiddleware "github.com/kakhavain/telemetry-tracker/internal/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/observability"
	"github.com/kakhavain/telemetry-tracker/internal/storage"
)

type mockStorer struct {
	StoreFunc func(ctx context.Context, event storage.Event) error
}

func (m *mockStorer) StoreEvent(ctx context.Context, event storage.Event) error {
	return m.StoreFunc(ctx, event)
}

func (m *mockStorer) Close() {}

func TestEventHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		contentType    string
		body           string
		storeErr       error
		expectedStatus int
	}{
		{
			name:        "Valid event",
			method:      http.MethodPost,
			contentType: "application/json",
			body: `{
				"event_type": "login",
				"timestamp": "2024-03-28T12:34:56Z",
				"data": {"user": "test"}
			}`,
			expectedStatus: http.StatusAccepted,
		},
		{
			name:           "Wrong content type",
			method:         http.MethodPost,
			contentType:    "text/plain",
			body:           `{"event_type": "login"}`,
			expectedStatus: http.StatusUnsupportedMediaType,
		},
		{
			name:           "Malformed JSON",
			method:         http.MethodPost,
			contentType:    "application/json",
			body:           `{"event_type": login"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing event_type",
			method:         http.MethodPost,
			contentType:    "application/json",
			body: `{
				"timestamp": "2024-03-28T12:34:56Z",
				"data": {"user": "test"}
			}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Storage error",
			method:         http.MethodPost,
			contentType:    "application/json",
			body: `{
				"event_type": "login",
				"timestamp": "2024-03-28T12:34:56Z",
				"data": {}
			}`,
			storeErr:       errors.New("db failure"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	obs, _ := observability.InitObservability("noop")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockStore := &mockStorer{
				StoreFunc: func(ctx context.Context, event storage.Event) error {
					return tc.storeErr
				},
			}

			reg, _ := metrics.NewRegistry(obs.Meter())
			handler := handlers.NewEventHandler(mockStore, reg, obs)

			req := httptest.NewRequest(tc.method, "/events", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			ctx := appmiddleware.WithLogger(req.Context(), logger)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			be.Equal(t, tc.expectedStatus, rec.Code)
		})
	}
}
