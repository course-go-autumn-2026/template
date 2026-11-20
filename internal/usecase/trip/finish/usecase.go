package finish

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
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

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	var result trip.Trip

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		currentTrip, err := u.tripRepo.GetTripForUpdate(txCtx, id)
		if err != nil {
			return fmt.Errorf("get trip for update: %w", err)
		}

		previousStatus := currentTrip.Status
		now := time.Now().UTC()

		if err := currentTrip.Finish(now); err != nil {
			return err
		}

		if err := u.tripRepo.Update(txCtx, currentTrip); err != nil {
			return fmt.Errorf("update trip: %w", err)
		}

		if err := u.tripHistoryRepo.AddTransition(txCtx, currentTrip.ID, &previousStatus, trip.StatusCompleted, nil); err != nil {
			return fmt.Errorf("save trip history: %w", err)
		}

		result = currentTrip
		return nil
	})
	if err != nil {
		return trip.Trip{}, fmt.Errorf("on transaction: %w", err)
	}

	if u.metrics != nil {
		u.metrics.TripCompleted(ctx)
	}

	return result, nil
}
