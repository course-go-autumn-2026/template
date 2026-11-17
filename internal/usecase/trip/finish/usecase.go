package finish

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
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

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var entity trip.Trip

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error

		entity, err = u.repository.GetTripForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("lock trip: %w", err)
		}

		previous := entity.Status

		if err := entity.Finish(u.clock.Now()); err != nil {
			return err
		}

		if err := u.repository.FinishTrip(ctx, id, *entity.FinishedAt); err != nil {
			return fmt.Errorf("save completed trip: %w", err)
		}

		return u.repository.AddStatusHistory(ctx, id, &previous, entity.Status, "trip finished")
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	u.observer.TripCompleted(ctx)
	u.logger.InfoContext(ctx, "trip completed", "trip_id", id)

	return entity, nil
}
