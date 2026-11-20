package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Metrics struct {
	httpRequestDuration      metric.Float64Histogram
	httpActiveRequests       metric.Int64UpDownCounter
	tripsCreated             metric.Int64Counter
	tripsCompleted           metric.Int64Counter
	staleChecks              metric.Int64Counter
	positionQueueSize        metric.Int64UpDownCounter
	positionRequests         metric.Int64Counter
	positionRequestsDuration metric.Float64Histogram
	activeWorkers            metric.Int64UpDownCounter
	positionsSaved           metric.Int64Counter
}

func NewMetrics() (*Metrics, error) {
	meter := otel.GetMeterProvider().Meter("trip-service")

	requestDuration, err := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("Duration of HTTP server requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create http.server.request.duration histogram: %w", err)
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create http.server.active_requests counter: %w", err)
	}

	tripsCreated, err := meter.Int64Counter(
		"tripgo.trips.created",
		metric.WithDescription("Total number of created trips"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.created counter: %w", err)
	}

	tripsCompleted, err := meter.Int64Counter(
		"tripgo.trips.completed",
		metric.WithDescription("Total number of completed trips"),
		metric.WithUnit("{trip}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.trips.completed counter: %w", err)
	}

	staleChecks, err := meter.Int64Counter(
		"tripgo.stale_checks",
		metric.WithDescription("Total number of stale trip checks by enqueue result"),
		metric.WithUnit("{check}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.stale_checks counter: %w", err)
	}

	positionQueueSize, err := meter.Int64UpDownCounter(
		"tripgo.position_requests.queue.size",
		metric.WithDescription("Current number of pending position requests in queue"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.position_requests.queue.size counter: %w", err)
	}

	positionRequests, err := meter.Int64Counter(
		"tripgo.position_requests",
		metric.WithDescription("Total number of position requests by result"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.position_requests counter: %w", err)
	}

	positionRequestsDuration, err := meter.Float64Histogram(
		"tripgo.position_requests.duration",
		metric.WithDescription("Duration of position requests"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.position_requests.duration histogram: %w", err)
	}

	activeWorkers, err := meter.Int64UpDownCounter(
		"tripgo.position_requests.workers.active",
		metric.WithDescription("Number of active workers processing position requests"),
		metric.WithUnit("{worker}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.position_requests.workers.active counter: %w", err)
	}

	positionsSaved, err := meter.Int64Counter(
		"tripgo.positions.saved",
		metric.WithDescription("Total number of saved positions"),
		metric.WithUnit("{position}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create tripgo.positions.saved counter: %w", err)
	}

	return &Metrics{
		httpRequestDuration:      requestDuration,
		httpActiveRequests:       activeRequests,
		tripsCreated:             tripsCreated,
		tripsCompleted:           tripsCompleted,
		staleChecks:              staleChecks,
		positionQueueSize:        positionQueueSize,
		positionRequests:         positionRequests,
		positionRequestsDuration: positionRequestsDuration,
		activeWorkers:            activeWorkers,
		positionsSaved:           positionsSaved,
	}, nil
}

func (m *Metrics) RecordHTTPRequest(ctx context.Context, method, route string, statusCode int, duration time.Duration) {
	if m == nil || m.httpRequestDuration == nil {
		return
	}
	m.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(route),
		semconv.HTTPResponseStatusCodeKey.Int(statusCode),
	))
}

func (m *Metrics) AddActiveRequest(ctx context.Context, method, route string, delta int64) {
	if m == nil || m.httpActiveRequests == nil {
		return
	}
	m.httpActiveRequests.Add(ctx, delta, metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.HTTPRouteKey.String(route),
	))
}

func (m *Metrics) TripCreated(ctx context.Context) {
	if m == nil || m.tripsCreated == nil {
		return
	}
	m.tripsCreated.Add(ctx, 1)
}

func (m *Metrics) TripCompleted(ctx context.Context) {
	if m == nil || m.tripsCompleted == nil {
		return
	}
	m.tripsCompleted.Add(ctx, 1)
}

func (m *Metrics) StaleCheck(ctx context.Context, result string) {
	if m == nil || m.staleChecks == nil {
		return
	}
	m.staleChecks.Add(ctx, 1, metric.WithAttributes(
		attribute.String("result", result),
	))
}

func (m *Metrics) AddPositionQueueSize(ctx context.Context, delta int64) {
	if m == nil || m.positionQueueSize == nil {
		return
	}
	m.positionQueueSize.Add(ctx, delta)
}

func (m *Metrics) PositionRequest(ctx context.Context, result string) {
	if m == nil || m.positionRequests == nil {
		return
	}
	m.positionRequests.Add(ctx, 1, metric.WithAttributes(
		attribute.String("result", result),
	))
}

func (m *Metrics) RecordPositionRequestDuration(ctx context.Context, duration time.Duration) {
	if m == nil || m.positionRequestsDuration == nil {
		return
	}
	m.positionRequestsDuration.Record(ctx, duration.Seconds())
}

func (m *Metrics) AddActiveWorker(ctx context.Context, delta int64) {
	if m == nil || m.activeWorkers == nil {
		return
	}
	m.activeWorkers.Add(ctx, delta)
}

func (m *Metrics) PositionSaved(ctx context.Context) {
	if m == nil || m.positionsSaved == nil {
		return
	}
	m.positionsSaved.Add(ctx, 1)
}
