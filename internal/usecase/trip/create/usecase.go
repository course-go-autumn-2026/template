package create

import (
	"context"
	"log/slog"
	"time"

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

func (u *Usecase) Execute(ctx context.Context, input model.CreateTripInput) (*model.Trip, error) {
	now := time.Now().UTC()

	tr, err := trip.NewTrip(
		input.UserID,
		input.DriverID,
		input.StartPoint,
		input.EndPoint,
		input.Price,
		now,
	)
	if err != nil {
		return nil, err
	}

	reason := "trip created"
	history := &model.TripStatusHistory{
		TripID:     tr.ID,
		FromStatus: nil,
		ToStatus:   string(trip.StatusActive),
		Reason:     &reason,
		ChangedAt:  now,
	}

	err = u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.repo.Create(txCtx, tr); err != nil {
			return err
		}
		if err := u.repo.CreateStatusHistory(txCtx, history); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if u.logger != nil {
		u.logger.InfoContext(ctx, "trip created", slog.String("trip_id", tr.ID.String()))
	}
	if u.metrics != nil {
		u.metrics.TripCreated(ctx)
	}

	return tr, nil
}
