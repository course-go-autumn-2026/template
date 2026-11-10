package trip

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/murmyauuu/template/api"
)

type TxManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	repository     *Repository
	history        *HistoryRepository
	txManager      TxManager
	idempotency    *IdempotencyRepository
	idempotencyTTL time.Duration
}

func NewService(repository *Repository, history *HistoryRepository, txManager TxManager, idempotency *IdempotencyRepository, ttl time.Duration) *Service {
	return &Service{repository: repository, history: history, txManager: txManager, idempotency: idempotency, idempotencyTTL: ttl}
}

func (s *Service) Create(ctx context.Context, data api.TripData, key *uuid.UUID) (api.Trip, bool, error) {
	var hash [sha256.Size]byte
	if key != nil {
		body, err := json.Marshal(data)
		if err != nil {
			return api.Trip{}, false, fmt.Errorf("encode idempotency request: %w", err)
		}
		hash = sha256.Sum256(body)
	}
	var created api.Trip
	fresh := true
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		if key != nil {
			var err error
			fresh, err = s.idempotency.Claim(ctx, *key, hash[:], time.Now().UTC().Add(s.idempotencyTTL))
			if err != nil {
				return err
			}
			if err := s.idempotency.Prune(ctx, *key); err != nil {
				return err
			}
			if !fresh {
				created, err = s.idempotency.Replay(ctx, *key, hash[:])
				return err
			}
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generate trip ID: %w", err)
		}
		value := api.Trip{
			Id: id, UserId: data.UserId, DriverId: data.DriverId,
			StartPoint: data.StartPoint, EndPoint: data.EndPoint, Price: data.Price,
			Status: api.Active, StartedAt: time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err = s.repository.Create(ctx, value)
		if err != nil {
			return err
		}
		if err := s.history.Append(ctx, created.Id, nil, api.Active, created.StartedAt); err != nil {
			return err
		}
		if key != nil {
			return s.idempotency.Save(ctx, *key, created)
		}
		return nil
	})
	if err != nil {
		return api.Trip{}, false, err
	}
	return created, fresh, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) Finish(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	var finished api.Trip
	err := s.txManager.Do(ctx, func(ctx context.Context) error {
		var err error
		finished, err = s.repository.Finish(ctx, id, finishedAt)
		if err != nil {
			return err
		}
		from := api.Active
		return s.history.Append(ctx, id, &from, api.Completed, finishedAt)
	})
	if err != nil {
		return api.Trip{}, err
	}
	return finished, nil
}
