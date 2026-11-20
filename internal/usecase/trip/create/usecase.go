// Package create implements the "create trip" usecase.
package create

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

type Input struct {
	UserID, DriverID     uuid.UUID
	StartPoint, EndPoint trip.Coordinates
	Price                int64
	IdempotencyKey       *uuid.UUID
	RequestHash          []byte
}

type Result struct {
	Trip    model.Trip
	Created bool
}

type Usecase struct {
	repo tripRepository
	tx   txManager
}

func New(repo tripRepository, tx txManager) *Usecase {
	return &Usecase{repo: repo, tx: tx}
}

// Execute creates a trip, or returns the one created by a matching idempotency key.
func (u *Usecase) Execute(ctx context.Context, in Input) (Result, error) {
	t, err := trip.NewTrip(uuid.New(), in.UserID, in.DriverID, in.StartPoint, in.EndPoint, in.Price, time.Now().UTC())
	if err != nil {
		return Result{}, err
	}

	m := toModel(t)
	result := Result{Trip: m}

	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if in.IdempotencyKey != nil {
			claimed, existingHash, existingTripID, err := u.repo.ClaimIdempotencyKey(ctx, *in.IdempotencyKey, in.RequestHash)
			if err != nil {
				return err
			}

			if !claimed {
				if !bytes.Equal(existingHash, in.RequestHash) {
					return trip.ErrIdempotencyConflict
				}

				result.Trip, err = u.repo.Get(ctx, existingTripID)

				return err
			}
		}

		if err := u.repo.Create(ctx, m); err != nil {
			return err
		}

		if err := u.repo.AddStatusHistory(ctx, m.ID, nil, string(trip.StatusActive), "trip created", m.StartedAt); err != nil {
			return err
		}

		if in.IdempotencyKey != nil {
			if err := u.repo.CompleteIdempotencyKey(ctx, *in.IdempotencyKey, m.ID); err != nil {
				return err
			}
		}

		result.Created = true

		return nil
	})

	if err != nil {
		return Result{}, err
	}

	return result, nil
}

func toModel(t trip.Trip) model.Trip {
	return model.Trip{
		ID:         t.ID,
		UserID:     t.UserID,
		DriverID:   t.DriverID,
		StartPoint: model.Coordinates(t.StartPoint),
		EndPoint:   model.Coordinates(t.EndPoint),
		Price:      t.Price,
		Status:     string(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
