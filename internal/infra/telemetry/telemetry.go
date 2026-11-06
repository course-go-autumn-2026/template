package telemetry

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"

	"github.com/toooriii/tori/internal/config"
)

type Provider struct {
	tracerShutdown func(context.Context) error
	meterShutdown  func(context.Context) error
}

func Init(ctx context.Context, cfg config.Telemetry, version string) (*Provider, error) {
	host, insecure, err := splitEndpoint(cfg.OTLPEndpoint)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(append(parseAttrs(cfg.ResourceAttrs),
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(version),
		)...),
	)
	if err != nil {
		return nil, fmt.Errorf("resource: %w", err)
	}

	traceOpts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithURLPath("/v1/traces"),
	}
	if insecure {
		traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
	}
	traceExp, err := otlptracehttp.New(ctx, traceOpts...)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler(cfg.Sampler)),
	)
	otel.SetTracerProvider(tp)

	metricOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(host),
		otlpmetrichttp.WithURLPath("/v1/metrics"),
	}
	if insecure {
		metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
	}
	metricExp, err := otlpmetrichttp.New(ctx, metricOpts...)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExp, metric.WithInterval(5*time.Second))),
	)
	otel.SetMeterProvider(mp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		tracerShutdown: tp.Shutdown,
		meterShutdown:  mp.Shutdown,
	}, nil
}

func (p *Provider) Shutdown(ctx context.Context) error {
	if err := p.tracerShutdown(ctx); err != nil {
		return fmt.Errorf("tracer shutdown: %w", err)
	}
	return p.meterShutdown(ctx)
}

func splitEndpoint(raw string) (host string, insecure bool, err error) {
	u, parseErr := url.Parse(raw)
	if parseErr == nil && u.Host != "" {
		return u.Host, u.Scheme != "https", nil
	}
	if strings.Contains(raw, ":") {
		return raw, true, nil
	}
	return "", false, fmt.Errorf("invalid OTLP endpoint %q", raw)
}

func parseAttrs(s string) []attribute.KeyValue {
	var out []attribute.KeyValue
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			out = append(out, attribute.String(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])))
		}
	}
	return out
}

func sampler(name string) sdktrace.Sampler {
	switch name {
	case "always_off":
		return sdktrace.NeverSample()
	case "always_on":
		return sdktrace.AlwaysSample()
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}
