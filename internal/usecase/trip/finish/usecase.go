package finish

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	tripDomain "github.com/ggwpfu9/courseAvito/internal/domain/trip"
	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type UseCase struct {
	repo TripRepository
	tx   TxManager
}

func New(repo TripRepository, tx TxManager) *UseCase {
	return &UseCase{repo: repo, tx: tx}
}

func (u *UseCase) Execute(ctx context.Context, id string) (*appmodel.Trip, error) {
	var result *appmodel.Trip

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		trip, err := u.repo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if err := tripDomain.FinishTrip(trip, time.Now()); err != nil {
			return err
		}

		if err := u.repo.Update(ctx, *trip); err != nil {
			return err
		}

		result = trip
		return nil
	})

	if err == nil {
		meter := otel.Meter("trip-service")
		counter, _ := meter.Int64Counter("tripgo.trips.completed", metric.WithDescription("Total completed trips"))
		counter.Add(ctx, 1)
	}

	return result, err
}