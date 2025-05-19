// internal/observability/observability.go
package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

const schemaName = "https://github.com/kakhavai/telemetry-tracker"

type Provider interface {
	Logger() *slog.Logger
	Tracer() trace.Tracer
	Meter() metric.Meter
	Shutdown(ctx context.Context) error
}

type ObservabilityProvider struct {
	logger   *slog.Logger
	tracer   trace.Tracer
	meter    metric.Meter
	shutdown func(context.Context) error
}

func (o *ObservabilityProvider) Logger() *slog.Logger {
	return o.logger
}

func (o *ObservabilityProvider) Tracer() trace.Tracer {
	return o.tracer
}

func (o *ObservabilityProvider) Meter() metric.Meter {
	return o.meter
}

func (o *ObservabilityProvider) Shutdown(ctx context.Context) error {
	return o.shutdown(ctx)
}

// buildRootLogger installs a JSON handler at the requested level,
// makes it the slog process default, and returns it.
func buildRootLogger(level slog.Leveler) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	root := slog.New(h)
	slog.SetDefault(root)
	return root
}

func InitObservability(mode string) (Provider, error) {
	switch mode {

	case "otel":
		shutdown, err := SetupOTelSDK(context.Background())
		if err != nil {
			return nil, err
		}
		root := buildRootLogger(slog.LevelInfo)
		return &ObservabilityProvider{
			logger:   root.With("env", "otel"),
			tracer:   otel.Tracer(schemaName),
			meter:    otel.GetMeterProvider().Meter(schemaName),
			shutdown: shutdown,
		}, nil

	case "debug":
		root := buildRootLogger(slog.LevelDebug)
		return &ObservabilityProvider{
			logger:   root.With("env", "debug", "debug", true),
			tracer:   noop.NewTracerProvider().Tracer("debug"),
			meter:    otel.GetMeterProvider().Meter("debug"),
			shutdown: func(context.Context) error { return nil },
		}, nil

	case "local":
		root := buildRootLogger(slog.LevelInfo)
		return &ObservabilityProvider{
			logger:   root.With("env", "local"),
			tracer:   noop.NewTracerProvider().Tracer("local"),
			meter:    otel.GetMeterProvider().Meter("local"),
			shutdown: func(context.Context) error { return nil },
		}, nil

	case "noop":
		root := buildRootLogger(slog.LevelWarn)
		return &ObservabilityProvider{
			logger:   root.With("env", "noop"),
			tracer:   noop.NewTracerProvider().Tracer("noop"),
			meter:    otel.GetMeterProvider().Meter("noop"),
			shutdown: func(context.Context) error { return nil },
		}, nil

	default:
		return nil, fmt.Errorf("unsupported observability mode: %s", mode)
	}
}

// TODO: This was taken from https://github.com/grafana/docker-otel-lgtm, review what is actually needed
func SetupOTelSDK(ctx context.Context) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	shutdown := func(ctx context.Context) error {
		var combinedErr error
		for _, fn := range shutdownFuncs {
			combinedErr = errors.Join(combinedErr, fn(ctx))
		}
		shutdownFuncs = nil
		return combinedErr
	}

	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("otel-collector:4318"),
		otlptracehttp.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter))
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint("otel-collector:4318"),
		otlpmetrichttp.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)))
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpoint("otel-collector:4318"),
		otlploghttp.WithInsecure(),
	)

	if err != nil {
		return nil, err
	}
	loggerProvider := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)))
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	global.SetLoggerProvider(loggerProvider)

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		otel.Handle(err)
	}

	return shutdown, nil
}
