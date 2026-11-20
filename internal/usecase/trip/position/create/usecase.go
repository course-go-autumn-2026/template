package create

import (
	"context"
	"fmt"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/model"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

type Usecase struct {
	positionRepo tripPositionRepository
	tripRepo     tripRepository
	txManager    txManager
	metrics      metricsRecorder
	queue        *positions.Queue
}

func New(
	tripRepo tripRepository,
	positionRepo tripPositionRepository,
	txManager txManager,
	metrics metricsRecorder,
	queue *positions.Queue,
) *Usecase {
	return &Usecase{
		positionRepo: positionRepo,
		tripRepo:     tripRepo,
		txManager:    txManager,
		metrics:      metrics,
		queue:        queue,
	}
}

func (u *Usecase) Execute(ctx context.Context, data model.CreatePosition) (*trip.Position, error) {
	newPosition := trip.Position{
		ID:     data.ID,
		TripID: data.TripID,
		Coordinates: trip.Coordinates{
			Latitude:  data.Latitude,
			Longitude: data.Longitude,
		},
		RecordedAt: data.RecordedAt,
	}
	var position *trip.Position
	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		currentTrip, err := u.tripRepo.GetTrip(txCtx, data.TripID)
		if err != nil {
			return fmt.Errorf("get trip id: %w", err)
		}

		if currentTrip.Status == trip.StatusCompleted {
			return trip.ErrTripCompleted
		}

		position, err = u.positionRepo.AddPosition(txCtx, &newPosition)
		if err != nil {
			return fmt.Errorf("failed to add position: %w", err)
		}

		if err := u.positionRepo.UpdateLastPositionAt(txCtx, data.TripID, newPosition.RecordedAt); err != nil {
			return fmt.Errorf("failed to update last position: %w", err)
		}
		if u.queue != nil {
			u.queue.Delete(data.TripID)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("on transaction: %w", err)
	}

	if u.metrics != nil {
		u.metrics.PositionSaved(ctx)
	}

	return position, nil
}
