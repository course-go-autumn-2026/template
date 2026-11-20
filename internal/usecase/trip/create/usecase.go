package create

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/model"
)

type Usecase struct {
	tripRepo        tripRepository
	tripHistoryRepo tripHistoryRepository
	txManager       txManager
	metrics         metricsRecorder
}

func New(
	tripRepo tripRepository,
	tripHistoryRepo tripHistoryRepository,
	txManager txManager,
	metrics metricsRecorder,
) *Usecase {
	return &Usecase{
		tripRepo:        tripRepo,
		tripHistoryRepo: tripHistoryRepo,
		txManager:       txManager,
		metrics:         metrics,
	}
}

func (u *Usecase) Execute(ctx context.Context, data model.CreateTripData) (trip.Trip, error) {
	newTrip := trip.Trip{
		ID:         uuid.New(),
		UserID:     data.UserID,
		DriverID:   data.DriverID,
		StartPoint: data.StartPoint,
		EndPoint:   data.EndPoint,
		Price:      data.Price,
		Status:     trip.StatusActive,
		StartedAt:  time.Now().UTC(),
		FinishedAt: nil,
	}

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.tripRepo.Save(txCtx, newTrip); err != nil {
			return fmt.Errorf("save trip: %w", err)
		}

		if err := u.tripHistoryRepo.AddTransition(txCtx, newTrip.ID, nil, trip.StatusActive, nil); err != nil {
			return fmt.Errorf("save trip history: %w", err)
		}

		return nil
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("on transaction: %w", err)
	}

	if u.metrics != nil {
		u.metrics.TripCreated(ctx)
	}

	return newTrip, nil
}
