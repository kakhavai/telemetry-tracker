// internal/middleware/middleware.go

package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StructuredLogger creates a slog logger instance with JSON output.
func StructuredLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       level,
		AddSource:   false, // Set true for local debugging if needed (adds file:line)
		ReplaceAttr: nil,   // Can be used to customize attribute output
	}))
}

// --- Context Key ---
type loggerKey struct{}

// LoggerMiddleware injects the logger into the request context and logs request completion.
func LoggerMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := middleware.GetReqID(r.Context()) // Get Request ID from chi middleware

			// Create logger entry specific to this request
			requestLogger := logger.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", requestID),
			)

			// Inject logger into context
			ctx := context.WithValue(r.Context(), loggerKey{}, requestLogger)
			// Capture status code and bytes written for logging
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			start := time.Now()
			defer func() {
				duration := time.Since(start)
				status := ww.Status()
				// Note: BytesWritten() can be misleading if compression middleware is used later
				bytesWritten := ww.BytesWritten()

				logFn := requestLogger.Info // Default to Info
				if status >= 500 {
					logFn = requestLogger.Error
				} else if status >= 400 {
					logFn = requestLogger.Warn
				}

				logFn("Request completed",
					slog.Int("status", status),
					slog.Int("bytes", bytesWritten),
					slog.Duration("duration", duration),
					slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0), // Add ms duration
				)
			}()

			// Call next handler with the wrapped writer and context containing logger
			next.ServeHTTP(ww, r.WithContext(ctx))
		})
	}
}

// MetricsMiddleware wraps promhttp instrumentation using dedicated metrics.
// UPDATED: Removed .MustCurryWith from duration and size handlers.
func MetricsMiddleware(
	reqCounter *prometheus.CounterVec,
	reqDuration *prometheus.HistogramVec,
	respSize *prometheus.HistogramVec,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Pass the metric vectors directly. The promhttp handlers are smart enough
		// to use the required labels ("code", "method", "path") if the vector has them.
		return promhttp.InstrumentHandlerCounter(reqCounter,
			promhttp.InstrumentHandlerDuration(reqDuration, // Pass HistogramVec directly
				promhttp.InstrumentHandlerResponseSize(respSize, // Pass HistogramVec directly
					next,
				),
			),
		)
	}
}

// RecovererMiddleware catches panics, logs them with stack trace, and returns a 500 error.
// It wraps the ResponseWriter to check if headers were already sent.
func RecovererMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap the response writer *before* calling the next handler.
			// This allows us to check its status in the defer function.
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
					// Get request-specific logger if available (if LoggerMiddleware ran)
					requestLogger := GetLoggerFromContext(r.Context())
					if requestLogger == nil {
						// Fallback if panic happened before LoggerMiddleware added context
						requestLogger = logger.With(slog.String("request_id", middleware.GetReqID(r.Context())))
					}

					requestLogger.Error("Panic recovered",
						slog.Any("panic_value", rvr),
						slog.String("stack", string(debug.Stack())),
					)

					// Check if WriteHeader has already been called by inspecting Status.
					// If status is still 0, it means WriteHeader was not called explicitly.
					// We can safely write the 500 error header and message.
					if ww.Status() == 0 {
						// Use the wrapped writer (ww) to send the error response
						http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					}
					// If ww.Status() is non-zero, WriteHeader was already called by the handler
					// or subsequent middleware. We MUST NOT call WriteHeader or Error again.
				}
			}()

			// Call the next handler in the chain with the wrapped writer.
			next.ServeHTTP(ww, r)
		})
	}
}

// GetLoggerFromContext retrieves the logger associated with the request from the context.
// Returns nil if no logger is found. Useful in handlers.
func GetLoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}
	return nil
}