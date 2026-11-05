package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
)

type Request struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint trip.Point
	EndPoint   trip.Point
	Price      int64
}

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

func (u *Usecase) Execute(ctx context.Context, req Request) (*model.Trip, error) {
	now := time.Now().UTC()
	tr, err := trip.NewTrip(req.UserID, req.DriverID, req.StartPoint, req.EndPoint, req.Price, now)
	if err != nil {
		return nil, err
	}

	err = u.txManager.Do(ctx, func(ctx context.Context) error {
		if err := u.tripRepo.Save(ctx, tr); err != nil {
			return err
		}

		history := &model.TripStatusHistory{
			TripID:    tr.ID,
			Status:    string(tr.Status),
			ChangedAt: now,
		}
		return u.tripRepo.InsertHistory(ctx, history)
	})
	if err != nil {
		return nil, err
	}

	return tr, nil
}
