package main

import (
	"context"
	"errors"
	"fmt"
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

	"log/slog"

	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize observability.
	obs, err := observability.InitObservability("otel")
	if err != nil {
		slog.Error("Failed to initialize observability", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := obs.Shutdown(context.Background()); err != nil {
			slog.Error("Failed to shutdown observability", "error", err)
		}
	}()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded")

	metricsRegistry, err := metrics.NewRegistry(obs.Meter())
	if err != nil {
		slog.Error("Failed to initialize metrics", "error", err)
		os.Exit(1)
	}

	slog.Info("metric type",
		"name", "EventsReceivedTotal",
		"type", fmt.Sprintf("%T", metricsRegistry.EventsReceivedTotal),
	)

	store, err := storage.NewPostgresStore(ctx, cfg.DSN, obs)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err, "dsn_details", "host="+cfg.DBHost)
		os.Exit(1)
	}
	defer store.Close()

	appRouter := chi.NewRouter()
	appRouter.Use(
		chimid.RequestID,
		chimid.RealIP,
		middleware.RequestTelemetry(obs.Logger(), metricsRegistry),
		middleware.TracingMiddleware(obs.Tracer()),
		chimid.Timeout(60*time.Second),
	)

	eventHandler := handlers.NewEventHandler(store, metricsRegistry, obs)
	healthHandler := handlers.NewHealthHandler(obs)
	appRouter.Post("/events", eventHandler.ServeHTTP)
	appRouter.Get("/healthz", healthHandler.ServeHTTP)

	otelHandler := otelhttp.NewHandler(appRouter, "telemetry-tracker-router")

	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      otelHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("Starting server", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server error", "error", err)
			cancel()
		}
	}()

	<-sigChan
	slog.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}
	slog.Info("Server gracefully stopped")
}
