package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/domainerrors"
	"github.com/imrtlDrifter/template/internal/models"
	"github.com/imrtlDrifter/template/internal/repositories"
)

type TripService struct {
	repos     *repositories.Repository
	txManager TransactionManager
}

func NewTripService(repos *repositories.Repository, txManager TransactionManager) *TripService {
	return &TripService{
		repos:     repos,
		txManager: txManager,
	}
}

func (s *TripService) Create(ctx context.Context, command *models.CreateTripCommand, idempotencyKey *uuid.UUID) (*models.Trip, bool, error) {
	if err := command.Validate(); err != nil {
		return nil, false, err
	}

	trip := &models.Trip{
		ID:         uuid.New(),
		UserID:     command.UserID,
		DriverID:   command.DriverID,
		StartPoint: command.StartPoint,
		EndPoint:   command.EndPoint,
		Price:      command.Price,
		Status:     models.TripStatusActive,
		StartedAt:  time.Now().UTC(),
	}

	created := false

	requestHash, err := createRequestHash(command)
	if err != nil {
		return nil, false, err
	}

	err = s.txManager.Do(ctx, func(ctx context.Context) error {
		if idempotencyKey != nil {
			if err := s.repos.Idempotency.DeleteExpired(ctx); err != nil {
				return err
			}

			reserved, err := s.repos.Idempotency.TryCreate(ctx, *idempotencyKey, requestHash, time.Now().UTC().Add(24*time.Hour))
			if err != nil {
				return err
			}

			if !reserved {
				record, err := s.repos.Idempotency.Get(ctx, *idempotencyKey)
				if err != nil {
					return err
				}

				if !bytes.Equal(record.RequestHash, requestHash) {
					return domainerrors.ErrIdempotencyConflict
				}

				if record.TripID == nil {
					return errors.New("idempotency record has no trip")
				}

				trip, err = s.repos.Trip.GetByID(ctx, *record.TripID)
				if err != nil {
					return err
				}

				created = false
				return nil
			}
		}

		if err := s.repos.Trip.Create(ctx, trip); err != nil {
			return err
		}

		history := &models.TripStatusHistory{
			TripID:   trip.ID,
			ToStatus: models.TripStatusActive,
		}
		if err := s.repos.StatusHistory.CreateStatusHistory(ctx, history); err != nil {
			return err
		}

		if idempotencyKey != nil {
			if err := s.repos.Idempotency.BindTrip(ctx, *idempotencyKey, trip.ID); err != nil {
				return err
			}
		}

		created = true
		return nil
	})
	if err != nil {
		if repositories.IsDriverBusyError(err) {
			return nil, false, domainerrors.ErrDriverBusy
		}

		return nil, false, err
	}

	return trip, created, nil
}

func (s *TripService) GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error) {
	trip, err := s.repos.Trip.GetByID(ctx, id)
	if err != nil {
		if repositories.IsNotFoundError(err) {
			return nil, domainerrors.ErrTripNotFound
		}
		return nil, err
	}

	return trip, nil
}

func (s *TripService) Finish(ctx context.Context, id uuid.UUID) (*models.Trip, error) {
	var trip *models.Trip
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		finishedAt := time.Now().UTC()
		var err error
		trip, err = s.repos.Trip.Finish(ctx, id, finishedAt)
		if err != nil {
			if !repositories.IsNotFoundError(err) {
				return err
			}

			_, err = s.repos.Trip.GetByID(ctx, id)
			if err != nil {
				if repositories.IsNotFoundError(err) {
					return domainerrors.ErrTripNotFound
				}
				return err
			}

			return domainerrors.ErrTripCompleted
		}

		active := models.TripStatusActive
		history := &models.TripStatusHistory{
			TripID:     trip.ID,
			FromStatus: &active,
			ToStatus:   models.TripStatusCompleted,
		}
		if err := s.repos.StatusHistory.CreateStatusHistory(ctx, history); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return trip, nil
}

func createRequestHash(command *models.CreateTripCommand) ([]byte, error) {
	data, err := json.Marshal(command)
	if err != nil {
		return nil, fmt.Errorf("marshal create trip request: %w", err)
	}

	sum := sha256.Sum256(data)
	return sum[:], nil
}
