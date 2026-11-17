package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/metric"
)

type Business struct {
	created   metric.Int64Counter
	completed metric.Int64Counter
}

func NewBusiness(provider metric.MeterProvider) (*Business, error) {
	meter := provider.Meter("tripgo.business")

	created, err := meter.Int64Counter("tripgo.trips.created")
	if err != nil {
		return nil, fmt.Errorf("created counter: %w", err)
	}

	completed, err := meter.Int64Counter("tripgo.trips.completed")
	if err != nil {
		return nil, fmt.Errorf("completed counter: %w", err)
	}

	return &Business{created: created, completed: completed}, nil
}

func (b *Business) TripCreated(ctx context.Context) { b.created.Add(ctx, 1) }

func (b *Business) TripCompleted(ctx context.Context) { b.completed.Add(ctx, 1) }
