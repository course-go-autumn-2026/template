package finish

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
)

type Usecase struct {
	repo      tripRepository
	txManager txManager
	logger    *slog.Logger
	metrics   metrics
}

func New(repo tripRepository, txManager txManager, logger *slog.Logger, metrics metrics) *Usecase {
	return &Usecase{
		repo:      repo,
		txManager: txManager,
		logger:    logger,
		metrics:   metrics,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	if id == uuid.Nil {
		return nil, trip.NewValidationError("id", "trip id cannot be empty")
	}

	var finishedTrip *model.Trip
	now := time.Now().UTC()
	reason := "trip finished"
	fromStatus := string(trip.StatusActive)

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		tr, err := u.repo.Finish(txCtx, id, now)
		if err != nil {
			return err
		}
		finishedTrip = tr

		history := &model.TripStatusHistory{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   string(trip.StatusCompleted),
			Reason:     &reason,
			ChangedAt:  now,
		}

		return u.repo.CreateStatusHistory(txCtx, history)
	})
	if err != nil {
		return nil, err
	}

	if u.logger != nil {
		u.logger.InfoContext(ctx, "trip completed", slog.String("trip_id", id.String()))
	}
	if u.metrics != nil {
		u.metrics.TripCompleted(ctx)
	}

	return finishedTrip, nil
}
