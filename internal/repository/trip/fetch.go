package trip

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/model"
)

func (r *RepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	query, args, err := r.sb.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude",
		"end_latitude", "end_longitude",
		"price", "status", "started_at",
		"finished_at", "last_position_at",
	).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	exec := transaction.GetExecutor(ctx, r.pool)
	t, err := scanTrip(exec.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domaintrip.ErrTripNotFound
		}
		return nil, err
	}

	return t, nil
}

func (r *RepositoryImpl) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error) {
	query, args, err := r.sb.Update("trips").
		Set("status", string(domaintrip.StatusCompleted)).
		Set("finished_at", finishedAt).
		Where(sq.Eq{"id": id, "status": string(domaintrip.StatusActive)}).
		Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at, last_position_at").
		ToSql()
	if err != nil {
		return nil, err
	}

	exec := transaction.GetExecutor(ctx, r.pool)
	t, err := scanTrip(exec.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := r.GetByID(ctx, id)
			if getErr != nil {
				return nil, getErr
			}
			if existing.Status == domaintrip.StatusCompleted {
				return nil, domaintrip.ErrTripCompleted
			}
			return nil, domaintrip.ErrTripNotFound
		}
		return nil, err
	}

	return t, nil
}
