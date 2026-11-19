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
	HTTPRequestDuration     metric.Float64Histogram
	HTTPActiveRequests      metric.Int64UpDownCounter
	DBOperationDuration     metric.Float64Histogram
	TripsCreated            metric.Int64Counter
	TripsCompleted          metric.Int64Counter
	StaleChecks             metric.Int64Counter
	PositionQueueSize       metric.Int64UpDownCounter
	PositionWorkersActive   metric.Int64UpDownCounter
	PositionRequests        metric.Int64Counter
	PositionRequestDuration metric.Float64Histogram
	PositionsSaved          metric.Int64Counter
}

func NewNoopMetrics() *Metrics {
	meter := noop.NewMeterProvider().Meter("noop")
	httpDur, _ := meter.Float64Histogram("http.server.request.duration")
	httpAct, _ := meter.Int64UpDownCounter("http.server.active_requests")
	dbDur, _ := meter.Float64Histogram("db.client.operation.duration")
	created, _ := meter.Int64Counter("tripgo.trips.created")
	completed, _ := meter.Int64Counter("tripgo.trips.completed")
	staleChecks, _ := meter.Int64Counter("tripgo.stale_checks")
	queueSize, _ := meter.Int64UpDownCounter("tripgo.position_requests.queue.size")
	workersActive, _ := meter.Int64UpDownCounter("tripgo.position_requests.workers.active")
	posRequests, _ := meter.Int64Counter("tripgo.position_requests")
	posReqDuration, _ := meter.Float64Histogram("tripgo.position_requests.duration")
	posSaved, _ := meter.Int64Counter("tripgo.positions.saved")

	return &Metrics{
		HTTPRequestDuration:     httpDur,
		HTTPActiveRequests:      httpAct,
		DBOperationDuration:     dbDur,
		TripsCreated:            created,
		TripsCompleted:          completed,
		StaleChecks:             staleChecks,
		PositionQueueSize:       queueSize,
		PositionWorkersActive:   workersActive,
		PositionRequests:        posRequests,
		PositionRequestDuration: posReqDuration,
		PositionsSaved:          posSaved,
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

	metrics, err := newMetricsFromMeter(mp.Meter(serviceName))
	if err != nil {
		return NewNoopMetrics(), mp, err
	}

	return metrics, mp, nil
}

func newMetricsFromMeter(meter metric.Meter) (*Metrics, error) {
	httpDur, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	httpAct, err := meter.Int64UpDownCounter("http.server.active_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	dbDur, err := meter.Float64Histogram("db.client.operation.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	created, err := meter.Int64Counter("tripgo.trips.created", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, err
	}
	completed, err := meter.Int64Counter("tripgo.trips.completed", metric.WithUnit("{trip}"))
	if err != nil {
		return nil, err
	}
	staleChecks, err := meter.Int64Counter("tripgo.stale_checks", metric.WithUnit("{check}"))
	if err != nil {
		return nil, err
	}
	queueSize, err := meter.Int64UpDownCounter("tripgo.position_requests.queue.size", metric.WithUnit("{task}"))
	if err != nil {
		return nil, err
	}
	workersActive, err := meter.Int64UpDownCounter("tripgo.position_requests.workers.active", metric.WithUnit("{worker}"))
	if err != nil {
		return nil, err
	}
	posRequests, err := meter.Int64Counter("tripgo.position_requests", metric.WithUnit("{request}"))
	if err != nil {
		return nil, err
	}
	posReqDuration, err := meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	posSaved, err := meter.Int64Counter("tripgo.positions.saved", metric.WithUnit("{position}"))
	if err != nil {
		return nil, err
	}

	return &Metrics{
		HTTPRequestDuration:     httpDur,
		HTTPActiveRequests:      httpAct,
		DBOperationDuration:     dbDur,
		TripsCreated:            created,
		TripsCompleted:          completed,
		StaleChecks:             staleChecks,
		PositionQueueSize:       queueSize,
		PositionWorkersActive:   workersActive,
		PositionRequests:        posRequests,
		PositionRequestDuration: posReqDuration,
		PositionsSaved:          posSaved,
	}, nil
}
