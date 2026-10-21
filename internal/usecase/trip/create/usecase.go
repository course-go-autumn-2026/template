package create

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

type Request struct {
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     domain.Coordinates
	EndPoint       domain.Coordinates
	Price          int64
	IdempotencyKey *uuid.UUID
	RequestHash    string
}

type Result struct {
	Trip     domain.Trip
	Replayed bool
}

type UseCase struct {
	repo   tripRepository
	tx     txManager
	metric creationRecorder
}

func New(repo tripRepository, tx txManager, metric creationRecorder) *UseCase {
	return &UseCase{repo: repo, tx: tx, metric: metric}
}

func (u *UseCase) Execute(ctx context.Context, req Request) (Result, error) {
	if err := req.StartPoint.Validate("start_point"); err != nil {
		return Result{}, err
	}
	if err := req.EndPoint.Validate("end_point"); err != nil {
		return Result{}, err
	}
	if req.Price < 0 {
		return Result{}, &domain.ValidationError{Field: "price", Reason: "must be non-negative"}
	}
	trip := domain.Trip{
		ID: uuid.New(), UserID: req.UserID, DriverID: req.DriverID,
		StartPoint: req.StartPoint, EndPoint: req.EndPoint, Price: req.Price, Status: domain.StatusActive,
	}
	var replayed bool
	err := u.tx.Do(ctx, func(txCtx context.Context) error {
		if req.IdempotencyKey != nil {
			record, created, err := u.repo.ReserveIdempotencyKey(txCtx, *req.IdempotencyKey, req.RequestHash)
			if err != nil {
				return err
			}
			if !created {
				if record.RequestHash != req.RequestHash {
					return domain.ErrIdempotencyConflict
				}
				trip, err = u.repo.Get(txCtx, record.TripID)
				if err != nil {
					return err
				}
				replayed = true
				return nil
			}
		}
		var err error
		trip, err = u.repo.Create(txCtx, trip)
		if err != nil {
			return err
		}
		if err := u.repo.AddStatusHistory(txCtx, trip.ID, nil, domain.StatusActive, "trip created"); err != nil {
			return err
		}
		if req.IdempotencyKey != nil {
			return u.repo.SetIdempotencyTrip(txCtx, *req.IdempotencyKey, trip.ID)
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	if !replayed && u.metric != nil {
		u.metric.RecordCreated(ctx)
	}
	return Result{Trip: trip, Replayed: replayed}, nil
}
