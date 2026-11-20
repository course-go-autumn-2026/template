package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/George1487/MalyginGeorge/internal/config"
)

func Init(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	if cfg.OTelExporterOTLPEndpoint == "" {
		return func(context.Context) error { return nil }, nil
	}

	res, err := buildResource(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("build otel resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(parseSampler(cfg.OTelTracesSampler)),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, fmt.Errorf("create otlp metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	shutdown := func(shutdownCtx context.Context) error {
		var errs []error
		if err := tp.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown tracer provider: %w", err))
		}
		if err := mp.Shutdown(shutdownCtx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown meter provider: %w", err))
		}
		return errors.Join(errs...)
	}

	return shutdown, nil
}

func buildResource(ctx context.Context, cfg *config.Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.OTelServiceName),
	}

	if cfg.OTelResourceAttributes != "" {
		pairs := strings.Split(cfg.OTelResourceAttributes, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
			if len(kv) == 2 && kv[0] != "" {
				attrs = append(attrs, attribute.String(kv[0], kv[1]))
			}
		}
	}

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithHost(),
	)
}

func parseSampler(sampler string) sdktrace.Sampler {
	switch strings.ToLower(strings.TrimSpace(sampler)) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_always_on":
		fallthrough
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}
