package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)


type loggerKey struct{}

// RequestTelemetry middleware injects a logger into context, records logs, and metrics.
func RequestTelemetry(logger *slog.Logger, m *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context())
			reqLogger := logger.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", requestID),
			)

			ctx := context.WithValue(r.Context(), loggerKey{}, reqLogger)
			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			start := time.Now()
			next.ServeHTTP(rr, r.WithContext(ctx))
			duration := time.Since(start)

			status := rr.status
			bytes := rr.size
			attrs := []attribute.KeyValue{
				attribute.String("method", r.Method),
				attribute.String("status", strconv.Itoa(status)),
			}
			m.HTTPRequestTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
			m.RequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
			m.ResponseSizeBytes.Record(ctx, int64(bytes), metric.WithAttributes(attrs...))

			logFn := reqLogger.Info
			if status >= 500 {
				logFn = reqLogger.Error
			} else if status >= 400 {
				logFn = reqLogger.Warn
			}
			logFn("Request completed",
				slog.Int("status", status),
				slog.Int("bytes", bytes),
				slog.Duration("duration", duration),
			)
		})
	}
}

// TracingMiddleware starts a span for each request using the observability tracer.
func TracingMiddleware(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), "HTTP "+r.Method+" "+r.URL.Path)
			defer span.End()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	n, err := rr.ResponseWriter.Write(b)
	rr.size += n
	return n, err
}

func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return nil
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}
