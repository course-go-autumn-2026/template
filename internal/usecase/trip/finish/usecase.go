package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/model"
)

type UseCase struct {
	repo      tripRepository
	txManager txManager
	logger    logger
	metrics   metrics
}

func New(repo tripRepository, txManager txManager, logger logger, metrics metrics) *UseCase {
	return &UseCase{
		repo:      repo,
		txManager: txManager,
		logger:    logger,
		metrics:   metrics,
	}
}

func (uc *UseCase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	now := time.Now().UTC()
	var finishedTrip *model.Trip

	err := uc.txManager.Do(ctx, func(txCtx context.Context) error {
		existing, err := uc.repo.GetByID(txCtx, id)
		if err != nil {
			return err
		}

		domainT := domainTrip.Restore(
			existing.ID,
			existing.UserID,
			existing.DriverID,
			domainTrip.Coordinates{Latitude: existing.StartPoint.Latitude, Longitude: existing.StartPoint.Longitude},
			domainTrip.Coordinates{Latitude: existing.EndPoint.Latitude, Longitude: existing.EndPoint.Longitude},
			existing.Price,
			domainTrip.Status(existing.Status),
			existing.StartedAt,
			existing.FinishedAt,
			existing.CreatedAt,
			existing.UpdatedAt,
		)

		if err := domainT.Finish(now); err != nil {
			return err
		}

		t, err := uc.repo.Finish(txCtx, id, now)
		if err != nil {
			return err
		}
		finishedTrip = t

		fromStatus := model.StatusActive
		history := model.StatusHistory{
			TripID:     id,
			FromStatus: &fromStatus,
			ToStatus:   model.StatusCompleted,
			ChangedAt:  now,
		}
		return uc.repo.AddStatusHistory(txCtx, history)
	})
	if err != nil {
		return nil, err
	}

	if uc.logger != nil {
		uc.logger.InfoContext(ctx, "trip completed", "trip_id", finishedTrip.ID.String())
	}
	if uc.metrics != nil {
		uc.metrics.IncTripsCompleted(ctx)
	}

	return finishedTrip, nil
}
