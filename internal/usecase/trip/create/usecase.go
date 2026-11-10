package create

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type Input struct {
	UserId     uuid.UUID
	DriverId   uuid.UUID
	StartPoint domaintrip.Point
	EndPoint   domaintrip.Point
	Price      int64
}

type Output struct {
	Trip  domaintrip.Trip
	Fresh bool
}

type Usecase struct {
	repo           tripRepository
	tx             txManager
	history        historyRepository
	idempotency    idempotencyRepository
	idempotencyTTL time.Duration
	logger         *slog.Logger
	created        metric.Int64Counter
}

func New(repo tripRepository, tx txManager, history historyRepository, idempotency idempotencyRepository, ttl time.Duration, logger *slog.Logger) *Usecase {
	meter := otel.Meter("trip-service")
	created, _ := meter.Int64Counter("tripgo.trips.created")
	return &Usecase{repo: repo, tx: tx, history: history, idempotency: idempotency, idempotencyTTL: ttl, logger: logger, created: created}
}

func (u *Usecase) Execute(ctx context.Context, in Input, key *uuid.UUID) (Output, error) {
	var hash [sha256.Size]byte
	if key != nil {
		body, err := json.Marshal(in)
		if err != nil {
			return Output{}, fmt.Errorf("encode idempotency request: %w", err)
		}
		hash = sha256.Sum256(body)
	}
	var created domaintrip.Trip
	fresh := true
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		if key != nil {
			var err error
			fresh, err = u.idempotency.Claim(ctx, *key, hash[:], time.Now().UTC().Add(u.idempotencyTTL))
			if err != nil {
				return err
			}
			if err := u.idempotency.Prune(ctx, *key); err != nil {
				return err
			}
			if !fresh {
				created, err = u.idempotency.Replay(ctx, *key, hash[:])
				return err
			}
		}
		id, err := uuid.NewRandom()
		if err != nil {
			return fmt.Errorf("generate trip ID: %w", err)
		}
		value := domaintrip.Trip{
			Id:         id,
			UserId:     in.UserId,
			DriverId:   in.DriverId,
			StartPoint: in.StartPoint,
			EndPoint:   in.EndPoint,
			Price:      in.Price,
			Status:     domaintrip.StatusActive,
			StartedAt:  time.Now().UTC().Truncate(time.Microsecond),
		}
		created, err = u.repo.Create(ctx, value)
		if err != nil {
			return err
		}
		u.created.Add(ctx, 1)
		if u.logger != nil {
			u.logger.InfoContext(ctx, "trip created", "trip_id", created.Id.String())
		}
		if err := u.history.Append(ctx, created.Id, nil, domaintrip.StatusActive, created.StartedAt); err != nil {
			return err
		}
		if key != nil {
			return u.idempotency.Save(ctx, *key, created)
		}
		return nil
	})
	if err != nil {
		return Output{}, err
	}
	return Output{Trip: created, Fresh: fresh}, nil
}
