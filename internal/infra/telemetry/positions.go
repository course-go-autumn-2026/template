package telemetry

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Positions struct {
	saved    metric.Int64Counter
	checks   metric.Int64Counter
	requests metric.Int64Counter
	duration metric.Float64Histogram
	queue    metric.Int64ObservableGauge
	active   metric.Int64ObservableGauge
	meter    metric.Meter
}

func NewPositions(provider metric.MeterProvider) (*Positions, error) {
	p := &Positions{meter: provider.Meter("tripgo.positions")}

	for _, setting := range []struct {
		name   string
		target *metric.Int64Counter
	}{
		{"tripgo.positions.saved", &p.saved}, {"tripgo.stale_checks", &p.checks}, {"tripgo.position_requests", &p.requests},
	} {
		counter, err := p.meter.Int64Counter(setting.name)
		if err != nil {
			return nil, fmt.Errorf("position counter %s: %w", setting.name, err)
		}

		*setting.target = counter
	}

	var err error

	p.duration, err = p.meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("position duration: %w", err)
	}

	p.queue, err = p.meter.Int64ObservableGauge("tripgo.position_requests.queue.size")
	if err != nil {
		return nil, fmt.Errorf("position queue gauge: %w", err)
	}

	p.active, err = p.meter.Int64ObservableGauge("tripgo.position_requests.workers.active")
	if err != nil {
		return nil, fmt.Errorf("position worker gauge: %w", err)
	}

	return p, nil
}

func (p *Positions) ObservePool(snapshot func() (int64, int64)) (metric.Registration, error) {
	registration, err := p.meter.RegisterCallback(func(_ context.Context, observer metric.Observer) error {
		queued, active := snapshot()
		observer.ObserveInt64(p.queue, queued)
		observer.ObserveInt64(p.active, active)

		return nil
	}, p.queue, p.active)
	if err != nil {
		return nil, fmt.Errorf("position pool callback: %w", err)
	}

	return registration, nil
}

func (p *Positions) PositionSaved(ctx context.Context) { p.saved.Add(ctx, 1) }

func (p *Positions) StaleCheck(ctx context.Context, result string) {
	p.checks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

func (p *Positions) PositionRequest(ctx context.Context, result string, duration time.Duration) {
	attributes := metric.WithAttributes(attribute.String("result", result))
	p.requests.Add(ctx, 1, attributes)

	if result != "dropped" {
		p.duration.Record(ctx, duration.Seconds(), attributes)
	}
}
