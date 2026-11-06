package finish

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

var tripsCompleted, _ = otel.Meter("trip-service/usecase").Int64Counter(
	"tripgo.trips.completed",
	metric.WithUnit("{trip}"),
	metric.WithDescription("Completed trips"),
)

type UseCase struct {
	trips tripRepository
	tx    txManager
}

func New(trips tripRepository, tx txManager) *UseCase {
	return &UseCase{trips: trips, tx: tx}
}

func (uc *UseCase) Do(ctx context.Context, id uuid.UUID, now time.Time) (model.Trip, error) {
	var out model.Trip
	err := uc.tx.Do(ctx, func(ctx context.Context) error {
		got, err := uc.trips.Get(ctx, id)
		if err != nil {
			return err
		}
		domainTrip := got.ToDomain()
		if err := domainTrip.Finish(now); err != nil {
			return err
		}
		out, err = uc.trips.Finish(ctx, id, now)
		if err != nil {
			return err
		}
		from := string(trip.StatusActive)
		return uc.trips.AddHistory(ctx, out.ID, &from, string(trip.StatusCompleted), "trip finished")
	})

	if err == nil {
		tripsCompleted.Add(ctx, 1)
	}

	return out, err
}
