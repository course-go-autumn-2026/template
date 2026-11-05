package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type Usecase struct {
	tripRepo  tripRepository
	txManager txManager
}

func New(tripRepo tripRepository, txManager txManager) *Usecase {
	return &Usecase{
		tripRepo:  tripRepo,
		txManager: txManager,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	now := time.Now().UTC()
	var finished *model.Trip

	err := u.txManager.Do(ctx, func(ctx context.Context) error {
		tr, err := u.tripRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if err := tr.Finish(now); err != nil {
			return err
		}

		finished, err = u.tripRepo.Finish(ctx, id, now)
		if err != nil {
			return err
		}

		history := &model.TripStatusHistory{
			TripID:    id,
			Status:    string(finished.Status),
			ChangedAt: now,
		}
		return u.tripRepo.InsertHistory(ctx, history)
	})
	if err != nil {
		return nil, err
	}

	return finished, nil
}
