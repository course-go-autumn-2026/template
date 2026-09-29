package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"trip-service/internal/pkg/txmanager"
	"trip-service/internal/repository"
)

type TripService struct {
	repo *repository.TripRepository
	txm  txmanager.TxManager
}

func NewTripService(repo *repository.TripRepository, txm txmanager.TxManager) *TripService {
	return &TripService{
		repo: repo,
		txm:  txm,
	}
}

type CreateTripDTO struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
}

func (s *TripService) CreateTrip(ctx context.Context, dto CreateTripDTO) (*repository.Trip, error) {
	now := time.Now().UTC()
	trip := repository.Trip{
		ID:             uuid.New(),
		UserID:         dto.UserID,
		DriverID:       dto.DriverID,
		StartLatitude:  dto.StartLatitude,
		StartLongitude: dto.StartLongitude,
		EndLatitude:    dto.EndLatitude,
		EndLongitude:   dto.EndLongitude,
		Price:          dto.Price,
		Status:         "active",
		StartedAt:      now,
	}

	err := s.txm.Do(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, trip); err != nil {
			return err
		}
		return s.repo.AddStatusHistory(txCtx, trip.ID, trip.Status, now)
	})
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (*repository.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (*repository.Trip, error) {
	now := time.Now().UTC()
	var finishedTrip *repository.Trip

	// Атомарное обновление поездки и запись изменения статуса
	err := s.txm.Do(ctx, func(txCtx context.Context) error {
		trip, err := s.repo.Finish(txCtx, id, now)
		if err != nil {
			return err
		}
		if err := s.repo.AddStatusHistory(txCtx, id, "completed", now); err != nil {
			return err
		}
		finishedTrip = trip
		return nil
	})
	if err != nil {
		return nil, err
	}

	return finishedTrip, nil
}
