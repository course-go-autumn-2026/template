package create

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/model"
)

type Usecase struct {
	repository tripRepository
	tx         txManager
	clock      clock
	observer   observer
	logger     *slog.Logger
}

func New(repository tripRepository, tx txManager, clock clock, observer observer, logger *slog.Logger) *Usecase {
	return &Usecase{repository: repository, tx: tx, clock: clock, observer: observer, logger: logger}
}

func (u *Usecase) Execute(ctx context.Context, input model.CreateTripInput) (trip.Trip, error) {
	entity, err := trip.New(uuid.New(), input, u.clock.Now())
	if err != nil {
		return trip.Trip{}, err
	}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repository.CreateTrip(ctx, entity); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		return u.repository.AddStatusHistory(ctx, entity.ID, nil, trip.StatusActive, "trip created")
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	u.observer.TripCreated(ctx)
	u.logger.InfoContext(ctx, "trip created", "trip_id", entity.ID)

	return entity, nil
}
