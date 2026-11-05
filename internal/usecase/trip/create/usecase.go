package create

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type UseCase struct {
	repo TripRepository
	tx   TxManager
}

func New(repo TripRepository, tx TxManager) *UseCase {
	return &UseCase{
		repo: repo,
		tx:   tx,
	}
}

func (u *UseCase) Execute(ctx context.Context, trip appmodel.Trip) (*appmodel.Trip, error) {
	trip.ID = uuid.New().String()
	trip.Status = "active"
	trip.StartedAt = time.Now()

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		return u.repo.Save(ctx, trip)
	})
	if err != nil {
		return nil, err
	}

	meter := otel.Meter("trip-service")
	counter, _ := meter.Int64Counter("tripgo.trips.created", metric.WithDescription("Total created trips"))
	counter.Add(ctx, 1)

	return &trip, nil
}