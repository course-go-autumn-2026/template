package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Config struct {
	ServiceName string
	Version     string
	Endpoint    string
	Protocol    string
	Sampler     string
	LogLevel    slog.Level
}

func Setup(ctx context.Context, cfg Config) (*slog.Logger, func(context.Context) error, error) {
	var closers []func(context.Context) error
	shutdown := func(ctx context.Context) error {
		var result error
		for i := len(closers) - 1; i >= 0; i-- {
			result = errors.Join(result, closers[i](ctx))
		}
		return result
	}
	defer func() {
		// cleanup on error handled by caller via shutdown
	}()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", cfg.ServiceName),
			attribute.String("service.version", cfg.Version),
			attribute.String("service.namespace", "tripgo"),
			attribute.String("deployment.environment", "local"),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, shutdown, err
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, shutdown, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(time.Second)),
		sdktrace.WithSampler(samplerFrom(cfg.Sampler)),
	)
	closers = append(closers, tp.Shutdown)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, shutdown, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	closers = append(closers, mp.Shutdown)
	otel.SetMeterProvider(mp)

	otel.SetTextMapPropagator(propagation.TraceContext{})

	handler := newTraceHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	logger := slog.New(handler)

	return logger, shutdown, nil
}

func samplerFrom(name string) sdktrace.Sampler {
	switch name {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(0.1)
	default:
		return sdktrace.AlwaysSample()
	}
}
