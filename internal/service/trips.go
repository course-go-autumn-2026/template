package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/razrabbesezdav-art/template/internal/repository"
	"github.com/razrabbesezdav-art/template/internal/transaction"
)

type Trips struct {
	txManager *transaction.Manager
	trips     *repository.Trips
	history   *repository.History
}

func NewTrips(
	txManager *transaction.Manager,
	trips *repository.Trips,
	history *repository.History,
) *Trips {
	return &Trips{
		txManager: txManager,
		trips:     trips,
		history:   history,
	}
}

func (s *Trips) Create(
	ctx context.Context,
	data repository.TripData,
) (trip repository.Trip, err error) {
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		created, createErr := s.trips.Create(txCtx, data)
		if createErr != nil {
			return createErr
		}

		if historyErr := s.history.AddCreated(txCtx, created.ID); historyErr != nil {
			return historyErr
		}

		trip = created
		return nil
	})
	if err != nil {
		return repository.Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return trip, nil
}

func (s *Trips) Get(
	ctx context.Context,
	tripID uuid.UUID,
) (repository.Trip, error) {
	trip, err := s.trips.Get(ctx, tripID)
	if err != nil {
		return repository.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (s *Trips) Finish(
	ctx context.Context,
	tripID uuid.UUID,
) (trip repository.Trip, err error) {
	err = s.txManager.WithinTransaction(ctx, func(txCtx context.Context) error {
		finished, finishErr := s.trips.Finish(txCtx, tripID)
		if finishErr != nil {
			return finishErr
		}

		if historyErr := s.history.AddCompleted(txCtx, tripID); historyErr != nil {
			return historyErr
		}

		trip = finished
		return nil
	})
	if err != nil {
		return repository.Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	return trip, nil
}

func IsConflict(err error) bool {
	return errors.Is(err, repository.ErrDriverBusy) ||
		errors.Is(err, repository.ErrTripCompleted)
}
