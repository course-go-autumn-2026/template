package telemetry

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
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

func NewNoopMetrics() *Metrics {
	meter := noop.NewMeterProvider().Meter("noop")
	httpDur, _ := meter.Float64Histogram("http.server.request.duration")
	httpAct, _ := meter.Int64UpDownCounter("http.server.active_requests")
	dbDur, _ := meter.Float64Histogram("db.client.operation.duration")
	created, _ := meter.Int64Counter("tripgo.trips.created")
	completed, _ := meter.Int64Counter("tripgo.trips.completed")

	return &Metrics{
		HTTPRequestDuration: httpDur,
		HTTPActiveRequests:  httpAct,
		DBOperationDuration: dbDur,
		TripsCreated:        created,
		TripsCompleted:      completed,
	}
}

func InitMetrics(ctx context.Context, serviceName, endpoint, protocol string) (*Metrics, *sdkmetric.MeterProvider, error) {
	cleanEndpoint := strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")

	var (
		exporter sdkmetric.Exporter
		err      error
	)

	if strings.Contains(strings.ToLower(protocol), "grpc") {
		exporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(cleanEndpoint),
			otlpmetricgrpc.WithInsecure(),
		)
	} else {
		exporter, err = otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(cleanEndpoint),
			otlpmetrichttp.WithInsecure(),
		)
	}
	if err != nil {
		return NewNoopMetrics(), nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		return NewNoopMetrics(), nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	meter := mp.Meter(serviceName)

	httpDur, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	httpAct, err := meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	dbDur, err := meter.Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithUnit("{trip}"))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithUnit("{trip}"))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	return &Metrics{
		HTTPRequestDuration: httpDur,
		HTTPActiveRequests:  httpAct,
		DBOperationDuration: dbDur,
		TripsCreated:        created,
		TripsCompleted:      completed,
	}, mp, nil
}
