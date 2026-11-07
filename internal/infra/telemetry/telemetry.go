package telemetry

import (
	"context"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/XrystikLK/template/internal/config"
)

type Telemetry struct {
	Tracer   trace.Tracer
	Meter    metric.Meter
	Metrics  *Metrics
	shutdown func(context.Context) error
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t.shutdown != nil {
		return t.shutdown(ctx)
	}
	return nil
}

func parseResourceAttributes(raw string) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if raw == "" {
		return attrs
	}
	pairs := strings.Split(raw, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(kv) == 2 {
			attrs = append(attrs, attribute.String(kv[0], kv[1]))
		}
	}
	return attrs
}

func selectSampler(samplerType string) sdktrace.Sampler {
	switch strings.ToLower(strings.TrimSpace(samplerType)) {
	case "always_on":
		return sdktrace.AlwaysSample()
	case "always_off":
		return sdktrace.NeverSample()
	case "parentbased_always_off":
		return sdktrace.ParentBased(sdktrace.NeverSample())
	case "parentbased_always_on", "":
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

func Setup(ctx context.Context, cfg *config.Config) (*Telemetry, error) {
	serviceName := cfg.OtelServiceName
	if serviceName == "" {
		serviceName = "trip-service"
	}

	extraAttrs := parseResourceAttributes(cfg.OtelResourceAttributes)
	resAttrs := make([]attribute.KeyValue, 0, 3+len(extraAttrs))
	resAttrs = append(resAttrs,
		semconv.ServiceName(serviceName),
		attribute.String("service.namespace", "tripgo"),
		attribute.String("deployment.environment", "local"),
	)
	resAttrs = append(resAttrs, extraAttrs...)

	res, err := resource.New(ctx,
		resource.WithAttributes(resAttrs...),
		resource.WithSchemaURL(semconv.SchemaURL),
	)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	var (
		tracerProvider *sdktrace.TracerProvider
		meterProvider  *sdkmetric.MeterProvider
		shutdownFuncs  []func(context.Context) error
	)

	endpoint := cfg.OtelExporterOtlpEndpoint
	if endpoint != "" {
		isInsecure := strings.HasPrefix(endpoint, "http://")
		host := strings.TrimPrefix(endpoint, "http://")
		host = strings.TrimPrefix(host, "https://")
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(host),
		}
		if isInsecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		traceExporter, err := otlptracehttp.New(ctx, opts...)
		if err == nil {
			bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
			tracerProvider = sdktrace.NewTracerProvider(
				sdktrace.WithSampler(selectSampler(cfg.OtelTracesSampler)),
				sdktrace.WithResource(res),
				sdktrace.WithSpanProcessor(bsp),
			)
			otel.SetTracerProvider(tracerProvider)
			shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
		}

		metricOpts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(host),
		}
		if isInsecure {
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}
		metricExporter, err := otlpmetrichttp.New(ctx, metricOpts...)
		if err == nil {
			reader := sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(time.Second))
			meterProvider = sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(res),
				sdkmetric.WithReader(reader),
			)
			otel.SetMeterProvider(meterProvider)
			shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
		}
	}

	if tracerProvider == nil {
		tracerProvider = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
		otel.SetTracerProvider(tracerProvider)
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	}

	if meterProvider == nil {
		meterProvider = sdkmetric.NewMeterProvider(sdkmetric.WithResource(res))
		otel.SetMeterProvider(meterProvider)
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	}

	tracer := otel.Tracer(serviceName)
	meter := otel.Meter(serviceName)

	metrics, err := NewMetrics(meter)
	if err != nil {
		return nil, err
	}

	shutdown := func(shutdownCtx context.Context) error {
		var firstErr error
		for _, fn := range shutdownFuncs {
			if err := fn(shutdownCtx); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	}

	return &Telemetry{
		Tracer:   tracer,
		Meter:    meter,
		Metrics:  metrics,
		shutdown: shutdown,
	}, nil
}
