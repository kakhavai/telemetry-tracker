package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kakhavain/telemetry-tracker/internal/config"
	"github.com/kakhavain/telemetry-tracker/internal/handlers"
	"github.com/kakhavain/telemetry-tracker/internal/metrics"
	"github.com/kakhavain/telemetry-tracker/internal/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/observability"
	"github.com/kakhavain/telemetry-tracker/internal/storage"

	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// Create an OTEL-enabled logger (using otelslog, which bridges slog).
	logLevel := slog.LevelInfo
	if lvlStr := os.Getenv("LOG_LEVEL"); lvlStr != "" {
		var lvl slog.Level
		if err := lvl.UnmarshalText([]byte(lvlStr)); err == nil {
			logLevel = lvl
		} else {
			slog.Warn("Invalid LOG_LEVEL provided, using default INFO", slog.String("provided", lvlStr))
		}
	}
	logger := middleware.StructuredLogger(logLevel)
	slog.SetDefault(logger)

	// Create a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// --- Setup OpenTelemetry (tracing, metrics, logs) ---
	otelShutdown, err := observability.SetupOTelSDK(ctx)
	if err != nil {
		logger.Error("Failed to set up OpenTelemetry", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := otelShutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown OpenTelemetry", slog.Any("error", err))
		}
	}()

	// Initialize custom metrics registry.
	metricsRegistry, err := metrics.NewRegistry()
	if err != nil {
		logger.Error("Failed to initialize metrics", slog.Any("error", err))
		os.Exit(1)
	}

	// Set up storage.
	store, err := storage.NewPostgresStore(ctx, cfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database", slog.Any("error", err), slog.String("dsn_details", "host="+cfg.DBHost))
		os.Exit(1)
	}
	defer store.Close()

	// Configure router and middleware.
	appRouter := chi.NewRouter()
	appRouter.Use(
		chimid.RequestID,
		chimid.RealIP,
		middleware.RecordRequestMetrics(metricsRegistry),
		middleware.Logger(logger),
		// Wrap each request in a trace span (optional, as otelhttp also creates spans).
		middleware.TracingMiddleware(),
		chimid.Timeout(60*time.Second),
	)

	// Set up handlers.
	eventHandler := handlers.NewEventHandler(store, metricsRegistry)
	healthHandler := &handlers.HealthHandler{}
	appRouter.Post("/events", eventHandler.ServeHTTP)
	appRouter.Get("/healthz", healthHandler.ServeHTTP)

	// Wrap router with OTEL HTTP instrumentation.
	otelHandler := otelhttp.NewHandler(appRouter, "telemetry-tracker-router")

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      otelHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start the server.
	go func() {
		logger.Info("Starting server", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", slog.Any("error", err))
			cancel()
		}
	}()

	<-sigChan
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.Any("error", err))
	}
	logger.Info("Server gracefully stopped")
}
