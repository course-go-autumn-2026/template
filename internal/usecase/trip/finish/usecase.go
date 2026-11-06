package finish

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

type UseCase struct {
	repo   tripRepository
	tx     txManager
	metric completionRecorder
}

func New(repo tripRepository, tx txManager, metric completionRecorder) *UseCase {
	return &UseCase{repo: repo, tx: tx, metric: metric}
}

func (u *UseCase) Execute(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	var trip domain.Trip
	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		var err error
		trip, err = u.repo.Get(txCtx, id)
		if err != nil {
			return err
		}
		if err := trip.CanFinish(); err != nil {
			return err
		}
		trip, err = u.repo.Finish(txCtx, id)
		if err != nil {
			return err
		}
		from := domain.StatusActive
		return u.repo.AddStatusHistory(txCtx, id, &from, domain.StatusCompleted, "trip finished")
	})
	if err != nil {
		return domain.Trip{}, err
	}
	if u.metric != nil {
		u.metric.RecordCompleted(ctx)
	}
	return trip, nil
}
