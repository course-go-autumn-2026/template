package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/model"
)

type Request struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint model.Coordinates
	EndPoint   model.Coordinates
	Price      int64
}

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

func (uc *UseCase) Execute(ctx context.Context, req Request) (*model.Trip, error) {
	now := time.Now().UTC()

	domainEntity, err := domainTrip.New(
		req.UserID,
		req.DriverID,
		domainTrip.Coordinates{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		domainTrip.Coordinates{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		req.Price,
		now,
	)
	if err != nil {
		return nil, err
	}

	tripModel := model.Trip{
		ID:         domainEntity.ID,
		UserID:     domainEntity.UserID,
		DriverID:   domainEntity.DriverID,
		StartPoint: req.StartPoint,
		EndPoint:   req.EndPoint,
		Price:      domainEntity.Price,
		Status:     model.Status(domainEntity.Status),
		StartedAt:  domainEntity.StartedAt,
		CreatedAt:  domainEntity.CreatedAt,
		UpdatedAt:  domainEntity.UpdatedAt,
	}

	err = uc.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Save(txCtx, tripModel); err != nil {
			return err
		}

		history := model.StatusHistory{
			TripID:     tripModel.ID,
			FromStatus: nil,
			ToStatus:   model.StatusActive,
			ChangedAt:  now,
		}
		return uc.repo.AddStatusHistory(txCtx, history)
	})
	if err != nil {
		return nil, err
	}

	if uc.logger != nil {
		uc.logger.InfoContext(ctx, "trip created", "trip_id", tripModel.ID.String())
	}
	if uc.metrics != nil {
		uc.metrics.IncTripsCreated(ctx)
	}

	return &tripModel, nil
}
