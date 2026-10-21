package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

type Providers struct {
	Trace     *sdktrace.TracerProvider
	Metric    *sdkmetric.MeterProvider
	Created   metric.Int64Counter
	Completed metric.Int64Counter
}

func (p *Providers) RecordCreated(ctx context.Context)   { p.Created.Add(ctx, 1) }
func (p *Providers) RecordCompleted(ctx context.Context) { p.Completed.Add(ctx, 1) }

func New(ctx context.Context) (*Providers, error) {
	if strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "" {
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required")
	}
	service := os.Getenv("OTEL_SERVICE_NAME")
	if service == "" {
		return nil, fmt.Errorf("OTEL_SERVICE_NAME is required")
	}
	res, err := resource.New(ctx, resource.WithFromEnv(), resource.WithAttributes(
		attribute.String("service.name", service), attribute.String("service.version", "dev"),
	))
	if err != nil {
		return nil, fmt.Errorf("telemetry resource: %w", err)
	}
	var traceExporter sdktrace.SpanExporter
	var metricExporter sdkmetric.Exporter
	switch os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL") {
	case "http/protobuf":
		traceExporter, err = otlptracehttp.New(ctx)
		if err == nil {
			metricExporter, err = otlpmetrichttp.New(ctx)
		}
	case "grpc":
		traceExporter, err = otlptracegrpc.New(ctx)
		if err == nil {
			metricExporter, err = otlpmetricgrpc.New(ctx)
		}
	default:
		return nil, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf or grpc")
	}
	if err != nil {
		return nil, fmt.Errorf("telemetry exporter: %w", err)
	}
	var sampler sdktrace.Sampler
	switch os.Getenv("OTEL_TRACES_SAMPLER") {
	case "parentbased_always_on":
		sampler = sdktrace.ParentBased(sdktrace.AlwaysSample())
	case "always_on":
		sampler = sdktrace.AlwaysSample()
	case "always_off":
		sampler = sdktrace.NeverSample()
	default:
		return nil, fmt.Errorf("unsupported OTEL_TRACES_SAMPLER")
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithSampler(sampler), sdktrace.WithBatcher(traceExporter))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second))))
	meter := mp.Meter("trip-service")
	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, err
	}
	return &Providers{Trace: tp, Metric: mp, Created: created, Completed: completed}, nil
}

func (p *Providers) Shutdown(ctx context.Context) error {
	traceErr := p.Trace.Shutdown(ctx)
	metricErr := p.Metric.Shutdown(ctx)
	if traceErr != nil {
		return traceErr
	}
	return metricErr
}

func SpanContext(ctx context.Context) trace.SpanContext { return trace.SpanContextFromContext(ctx) }
