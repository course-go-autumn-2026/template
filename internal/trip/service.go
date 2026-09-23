package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	txm   TxManager
	repo  *Repository
	idTTL time.Duration
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

func NewService(txm TxManager, repo *Repository, idTTL time.Duration) *Service {
	return &Service{txm: txm, repo: repo, idTTL: idTTL}
}

func (s *Service) Create(ctx context.Context, in CreateInput, idempotencyKey *uuid.UUID, requestHash string) (Trip, bool, error) {
	var created Trip
	var replay bool

	err := s.txm.Do(ctx, func(ctx context.Context) error {
		if idempotencyKey != nil {
			if err := s.repo.LockIdempotencyKey(ctx, *idempotencyKey); err != nil {
				return err
			}
			rec, err := s.repo.FindIdempotency(ctx, *idempotencyKey, time.Now().UTC())
			if err != nil {
				return err
			}
			if rec != nil {
				if rec.RequestHash != requestHash {
					return ErrIdempotencyConflict
				}
				existing, err := s.repo.Get(ctx, rec.TripID)
				if err != nil {
					return err
				}
				created = existing
				replay = true
				return nil
			}
		}

		now := time.Now().UTC()
		created = Trip{
			ID:        uuid.New(),
			UserID:    in.UserID,
			DriverID:  in.DriverID,
			StartLat:  in.StartLat,
			StartLon:  in.StartLon,
			EndLat:    in.EndLat,
			EndLon:    in.EndLon,
			Price:     in.Price,
			Status:    StatusActive,
			StartedAt: now,
		}

		if err := s.repo.Create(ctx, created); err != nil {
			return err
		}
		if err := s.repo.InsertStatusHistory(ctx, created.ID, nil, StatusActive, "created"); err != nil {
			return err
		}
		if idempotencyKey != nil {
			err := s.repo.SaveIdempotency(ctx, IdempotencyRecord{
				Key:         *idempotencyKey,
				RequestHash: requestHash,
				TripID:      created.ID,
				ExpiresAt:   now.Add(s.idTTL),
			})
			if errors.Is(err, errUniqueIdempotency) {
				rec, findErr := s.repo.FindIdempotency(ctx, *idempotencyKey, time.Now().UTC())
				if findErr != nil {
					return findErr
				}
				if rec == nil {
					return ErrIdempotencyConflict
				}
				if rec.RequestHash != requestHash {
					return ErrIdempotencyConflict
				}
				existing, getErr := s.repo.Get(ctx, rec.TripID)
				if getErr != nil {
					return getErr
				}
				created = existing
				replay = true
				return nil
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return Trip{}, false, err
	}
	return created, replay, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	var finished Trip
	err := s.txm.Do(ctx, func(ctx context.Context) error {
		now := time.Now().UTC()
		t, err := s.repo.FinishActive(ctx, id, now)
		if err != nil {
			return err
		}
		from := StatusActive
		if err := s.repo.InsertStatusHistory(ctx, t.ID, &from, StatusCompleted, "finished"); err != nil {
			return err
		}
		finished = t
		return nil
	})
	if err != nil {
		return Trip{}, err
	}
	return finished, nil
}

func (s *Service) Ping(ctx context.Context) error {
	if err := s.repo.pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return nil
}
