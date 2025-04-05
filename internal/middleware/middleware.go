package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	"github.com/kakhavain/telemetry-tracker/internal/observability"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// StructuredLogger creates an OTEL-aware slog logger with JSON output.
func StructuredLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true, // enables source info for debugging
	}))
}

type loggerKey struct{}

// Logger middleware injects the logger into the request context and logs request completion.
func Logger(logger *slog.Logger) func(http.Handler) http.Handler {
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

			// Inject the logger into the request context.
			ctx := context.WithValue(r.Context(), loggerKey{}, reqLogger)
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				duration := time.Since(start)
				status := ww.Status()
				bytes := ww.BytesWritten()
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
			}()
			next.ServeHTTP(ww, r.WithContext(ctx))
		})
	}
}

// TracingMiddleware starts a span for each request using the observability tracer.
func TracingMiddleware(obs observability.Provider) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := obs.Tracer().Start(r.Context(), "HTTP "+r.Method+" "+r.URL.Path)
			defer span.End()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RecordRequestMetrics instruments HTTP request metrics.
func RecordRequestMetrics(m *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rr, r)
			duration := time.Since(start).Seconds()
			attrs := []attribute.KeyValue{
				attribute.String("method", r.Method),
				attribute.String("status", strconv.Itoa(rr.status)),
			}
			ctx := r.Context()
			m.HTTPRequestTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
			m.RequestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
			m.ResponseSizeBytes.Record(ctx, int64(rr.size), metric.WithAttributes(attrs...))
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
