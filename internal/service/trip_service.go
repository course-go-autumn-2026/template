package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/ggwpfu9/courseAvito/internal/domain"
	"github.com/ggwpfu9/courseAvito/internal/repository"
)

type TripService struct {
	repo *repository.TripRepo
	tx   repository.TxManager
}

func NewTripService(repo *repository.TripRepo, tx repository.TxManager) *TripService {
	return &TripService{repo: repo, tx: tx}
}

func (s *TripService) CreateTrip(ctx context.Context, trip *domain.Trip) error {
	trip.ID = uuid.New()
	trip.Status = "active"
	trip.StartedAt = time.Now()

	return s.tx.Do(ctx, func(ctx context.Context) error {
		return s.repo.Create(ctx, trip)
	})
}

func (s *TripService) GetTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TripService) FinishTrip(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	err := s.tx.Do(ctx, func(ctx context.Context) error {
		return s.repo.Finish(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}