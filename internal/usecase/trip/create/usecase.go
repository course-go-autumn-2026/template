package create

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

var tripsCreated, _ = otel.Meter("trip-service/usecase").Int64Counter(
	"tripgo.trips.created",
	metric.WithUnit("{trip}"),
	metric.WithDescription("Created trips"),
)

type Input struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	Start    model.Point
	End      model.Point
	Price    int64
	RawBody  []byte
	Key      *uuid.UUID
	Now      time.Time
}

type Output struct {
	Trip          model.Trip
	AlreadyExists bool
}

type UseCase struct {
	trips tripRepository
	idem  idempotencyRepository
	tx    txManager
}

func New(trips tripRepository, idem idempotencyRepository, tx txManager) *UseCase {
	return &UseCase{trips: trips, idem: idem, tx: tx}
}

func (uc *UseCase) Do(ctx context.Context, in Input) (Output, error) {
	domainTrip := trip.Trip{
		ID:        uuid.New(),
		UserID:    in.UserID,
		DriverID:  in.DriverID,
		Start:     trip.Point{Latitude: in.Start.Latitude, Longitude: in.Start.Longitude},
		End:       trip.Point{Latitude: in.End.Latitude, Longitude: in.End.Longitude},
		Price:     in.Price,
		Status:    trip.StatusActive,
		StartedAt: in.Now.UTC().Truncate(time.Microsecond),
	}
	if err := domainTrip.Validate(); err != nil {
		return Output{}, err
	}
	mt := model.FromDomain(domainTrip)

	if in.Key == nil {
		if err := uc.tx.Do(ctx, func(ctx context.Context) error {
			if err := uc.trips.Save(ctx, mt); err != nil {
				return err
			}
			return uc.trips.AddHistory(ctx, mt.ID, nil, string(trip.StatusActive), "trip created")
		}); err != nil {
			return Output{}, err
		}

		tripsCreated.Add(ctx, 1)
		return Output{Trip: mt}, nil
	}

	hash, err := hashBody(in.RawBody)
	if err != nil {
		return Output{}, err
	}

	createErr := uc.tx.Do(ctx, func(ctx context.Context) error {
		if err := uc.idem.Save(ctx, *in.Key, hash, mt.ID); err != nil {
			return err
		}
		if err := uc.trips.Save(ctx, mt); err != nil {
			return err
		}
		return uc.trips.AddHistory(ctx, mt.ID, nil, string(trip.StatusActive), "trip created")
	})

	switch {
	case createErr == nil:
		tripsCreated.Add(ctx, 1)
		return Output{Trip: mt}, nil

	case errors.Is(createErr, trip.ErrKeyAlreadyExists):
		rec, getErr := uc.idem.Get(ctx, *in.Key)
		if getErr != nil {
			return Output{}, getErr
		}
		if rec.RequestHash != hash {
			return Output{}, trip.ErrIdempotencyConflict
		}
		existing, getErr := uc.trips.Get(ctx, rec.TripID)
		if getErr != nil {
			return Output{}, getErr
		}
		return Output{Trip: existing, AlreadyExists: true}, nil

	default:
		return Output{}, createErr
	}
}

func hashBody(raw []byte) (string, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	canon, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}
