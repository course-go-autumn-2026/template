package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Metrics struct {
	HTTPRequestDuration metric.Float64Histogram
	HTTPActiveRequests  metric.Int64UpDownCounter
	DBOperationDuration metric.Float64Histogram
	TripsCreated        metric.Int64Counter
	TripsCompleted      metric.Int64Counter
}

func InitMetrics(ctx context.Context, serviceName, endpoint string) (*Metrics, *sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create metric exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create resource: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(serviceName)

	httpDur, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, nil, err
	}

	httpAct, err := meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, nil, err
	}

	dbDur, err := meter.Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, nil, err
	}

	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, nil, err
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, nil, err
	}

	return &Metrics{
		HTTPRequestDuration: httpDur,
		HTTPActiveRequests:  httpAct,
		DBOperationDuration: dbDur,
		TripsCreated:        created,
		TripsCompleted:      completed,
	}, mp, nil
}
