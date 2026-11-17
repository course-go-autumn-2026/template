package telemetry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"

	"github.com/freyliee/template/internal/config"
)

type Telemetry struct {
	Traces     *sdktrace.TracerProvider
	Metrics    *sdkmetric.MeterProvider
	Propagator propagation.TextMapPropagator
}

func New(ctx context.Context, cfg config.Config) (*Telemetry, error) {
	var te sdktrace.SpanExporter
	var me sdkmetric.Exporter
	var err error

	if cfg.OTLPProtocol == "grpc" {
		te, err = otlptracegrpc.New(ctx, otlptracegrpc.WithEndpointURL(cfg.OTLPEndpoint), otlptracegrpc.WithTimeout(cfg.ExportTimeout), otlptracegrpc.WithDialOption(grpc.WithDisableServiceConfig()))
	} else {
		te, err = otlptracehttp.New(ctx, otlptracehttp.WithEndpointURL(strings.TrimRight(cfg.OTLPEndpoint, "/")+"/v1/traces"), otlptracehttp.WithTimeout(cfg.ExportTimeout))
	}

	if err != nil {
		return nil, fmt.Errorf("trace exporter: %w", err)
	}

	if cfg.OTLPProtocol == "grpc" {
		me, err = otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithEndpointURL(cfg.OTLPEndpoint), otlpmetricgrpc.WithTimeout(cfg.ExportTimeout), otlpmetricgrpc.WithDialOption(grpc.WithDisableServiceConfig()))
	} else {
		me, err = otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(strings.TrimRight(cfg.OTLPEndpoint, "/")+"/v1/metrics"), otlpmetrichttp.WithTimeout(cfg.ExportTimeout))
	}

	if err != nil {
		return nil, errors.Join(fmt.Errorf("metric exporter: %w", err), te.Shutdown(ctx))
	}

	res := resource.NewSchemaless(attribute.String("service.name", "trip-service"), attribute.String("service.namespace", "tripgo"), attribute.String("service.version", cfg.ServiceVersion), attribute.String("deployment.environment", "local"))
	traces := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithSampler(sampler(cfg)), sdktrace.WithBatcher(te, sdktrace.WithExportTimeout(cfg.ExportTimeout)))
	metrics := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(me, sdkmetric.WithInterval(cfg.MetricExportInterval), sdkmetric.WithTimeout(cfg.ExportTimeout))))

	return &Telemetry{Traces: traces, Metrics: metrics, Propagator: propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})}, nil
}

func sampler(cfg config.Config) sdktrace.Sampler {
	var base sdktrace.Sampler

	switch cfg.TracesSampler {
	case "always_off", "parentbased_always_off":
		base = sdktrace.NeverSample()
	case "traceidratio", "parentbased_traceidratio":
		base = sdktrace.TraceIDRatioBased(cfg.SamplerArg)
	default:
		base = sdktrace.AlwaysSample()
	}

	switch cfg.TracesSampler {
	case "parentbased_always_on", "parentbased_always_off", "parentbased_traceidratio":
		return sdktrace.ParentBased(base)
	default:
		return base
	}
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	return errors.Join(t.Traces.Shutdown(ctx), t.Metrics.Shutdown(ctx))
}
