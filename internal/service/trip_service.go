package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/poiuyttq/tripgo/internal/model"
	"github.com/poiuyttq/tripgo/internal/repository"
	"github.com/poiuyttq/tripgo/internal/txmanager"
)

var (
	ErrDriverHasActiveTrip  = repository.ErrDriverHasActiveTrip
	ErrTripNotFound         = errors.New("trip not found")
	ErrTripAlreadyCompleted = errors.New("trip already completed")
)

type TripService struct {
	tripRepo        *repository.TripRepository
	tripHistoryRepo *repository.TripStatusHistoryRepository
	txManager       txmanager.TxManager
}

func NewTripService(tripRepo *repository.TripRepository, tripHistoryRepo *repository.TripStatusHistoryRepository, txManager txmanager.TxManager) *TripService {
	return &TripService{
		tripRepo:        tripRepo,
		tripHistoryRepo: tripHistoryRepo,
		txManager:       txManager,
	}
}

func (s *TripService) CreateTrip(ctx context.Context, trip model.Trip) (model.Trip, error) {
	var createdTrip model.Trip

	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		_, exists, err := s.tripRepo.GetActiveByDriverID(ctx, trip.DriverID)
		if err != nil {
			return fmt.Errorf("check active trip: %w", err)
		}
		if exists {
			return ErrDriverHasActiveTrip
		}

		if err := s.tripRepo.Create(ctx, trip); err != nil {
			return err
		}

		createdTrip = trip
		history := model.TripStatusHistory{
			TripID:     trip.ID,
			FromStatus: nil,
			ToStatus:   trip.Status,
			Reason:     nil,
			ChangedAt:  time.Now().UTC(),
		}
		if err := s.tripHistoryRepo.Add(ctx, history); err != nil {
			return fmt.Errorf("create trip status history: %w", err)
		}
		return nil
	})
	if err != nil {
		return model.Trip{}, err
	}

	return createdTrip, nil
}

func (s *TripService) GetTrip(ctx context.Context, tripID string) (model.Trip, error) {
	trip, err := s.tripRepo.GetByID(ctx, tripID)
	if err != nil {
		return model.Trip{}, err
	}
	return trip, nil
}

func (s *TripService) FinishTrip(ctx context.Context, tripID string) error {
	return s.txManager.Do(ctx, func(ctx context.Context) error {
		trip, err := s.tripRepo.GetByID(ctx, tripID)
		if err != nil {
			return ErrTripNotFound
		}
		if trip.Status == "completed" {
			return ErrTripAlreadyCompleted
		}

		finishedAt := time.Now().UTC()
		if err := s.tripRepo.Finish(ctx, tripID, finishedAt); err != nil {
			return fmt.Errorf("finish trip: %w", err)
		}

		history := model.TripStatusHistory{
			TripID:     trip.ID,
			FromStatus: &trip.Status,
			ToStatus:   "completed",
			Reason:     nil,
			ChangedAt:  finishedAt,
		}
		if err := s.tripHistoryRepo.Add(ctx, history); err != nil {
			return fmt.Errorf("write trip status history: %w", err)
		}

		return nil
	})
}
