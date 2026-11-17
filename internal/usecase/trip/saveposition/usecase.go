package saveposition

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type Usecase struct {
	repository tripRepository
	tx         txManager
	clock      clock
	canceller  canceller
	observer   observer
}

func New(repository tripRepository, tx txManager, clock clock, canceller canceller, observer observer) *Usecase {
	return &Usecase{
		repository: repository,
		tx:         tx,
		clock:      clock,
		canceller:  canceller,
		observer:   observer,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID, data trip.PositionData) (trip.Position, error) {
	if err := data.Validate(u.clock.Now()); err != nil {
		return trip.Position{}, err
	}

	data.RecordedAt = data.RecordedAt.UTC()

	var position trip.Position

	err := u.tx.Do(ctx, func(ctx context.Context) error {
		entity, err := u.repository.GetTripForUpdate(ctx, id)
		if err != nil {
			return fmt.Errorf("lock trip: %w", err)
		}

		if entity.Status != trip.StatusActive {
			return trip.ErrTripCompleted
		}

		position, err = u.repository.SavePosition(ctx, id, data)

		return err
	})
	if err != nil {
		return trip.Position{}, fmt.Errorf("save trip position: %w", err)
	}

	u.canceller.Cancel(id)
	u.observer.PositionSaved(ctx)

	return position, nil
}
