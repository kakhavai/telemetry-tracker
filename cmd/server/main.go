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
	appmetrics "github.com/kakhavain/telemetry-tracker/internal/metrics" // Aliased to avoid conflict
	"github.com/kakhavain/telemetry-tracker/internal/middleware"
	"github.com/kakhavain/telemetry-tracker/internal/storage"

	"github.com/go-chi/chi/v5"
	chimid "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// --- Configuration ---
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	// --- Structured Logger ---
	logLevel := slog.LevelInfo
	if levelStr := os.Getenv("LOG_LEVEL"); levelStr != "" {
		var level slog.Level
		if err := level.UnmarshalText([]byte(levelStr)); err == nil {
			logLevel = level
		} else {
			slog.Warn("Invalid LOG_LEVEL provided, using default INFO", slog.String("provided", levelStr))
		}
	}
	logger := middleware.StructuredLogger(logLevel)
	slog.SetDefault(logger) // Set globally

	// --- Setup Context for graceful shutdown ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// --- Dependencies ---
	metricsRegistry := appmetrics.NewRegistry()
	store, err := storage.NewPostgresStore(ctx, cfg.DSN)
	if err != nil {
		logger.Error("Failed to connect to database", slog.Any("error", err), slog.String("dsn_details", "host="+cfg.DBHost)) // Don't log full DSN with password
		os.Exit(1)
	}
	defer store.Close()

	// --- Create a separate router for app routes ---
	appRouter := chi.NewRouter()
	appRouter.Use(chimid.RequestID)
	appRouter.Use(chimid.RealIP)
	appRouter.Use(middleware.RecovererMiddleware(logger))
	appRouter.Use(middleware.MetricsMiddleware(
		metricsRegistry.HTTPRequestTotal, // Pass the request counter
		metricsRegistry.RequestDuration,  // Pass the duration histogram
		metricsRegistry.ResponseSize,     // Pass the response size histogram
	))
	appRouter.Use(middleware.LoggerMiddleware(logger))
	appRouter.Use(chimid.Timeout(60 * time.Second))

	// --- HTTP Handlers for app routes ---
	eventHandler := handlers.NewEventHandler(store, metricsRegistry)
	healthHandler := &handlers.HealthHandler{}
	appRouter.Post("/events", eventHandler.ServeHTTP)
	appRouter.Get("/healthz", healthHandler.ServeHTTP)

	// --- Main router: Mount app routes and add /metrics separately ---
	mainRouter := chi.NewRouter()
	mainRouter.Mount("/", appRouter)
	// Expose /metrics without wrapping it in our MetricsMiddleware
	mainRouter.Handle("/metrics", promhttp.HandlerFor(metricsRegistry.Reg, promhttp.HandlerOpts{}))

	// --- HTTP Server ---
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      mainRouter,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// --- Start Server Goroutine ---
	go func() {
		logger.Info("Starting server", slog.String("address", server.Addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Server error", slog.Any("error", err))
			cancel()
		}
	}()

	// --- Graceful Shutdown ---
	<-sigChan
	logger.Info("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", slog.Any("error", err))
	}

	logger.Info("Server gracefully stopped")
}
