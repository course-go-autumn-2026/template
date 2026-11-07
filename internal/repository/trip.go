package repository

import (
	"context"
	"errors"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/txmanager"
)

type TripRepository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *TripRepository) Create(ctx context.Context, trip *model.Trip) error {
	query, args, err := r.sb.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
			"finished_at", "last_position_at",
		).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartPoint.Latitude, trip.StartPoint.Longitude,
			trip.EndPoint.Latitude, trip.EndPoint.Longitude,
			trip.Price, trip.Status, trip.StartedAt,
			trip.FinishedAt, trip.LastPositionAt,
		).
		ToSql()
	if err != nil {
		return err
	}

	exec := txmanager.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return model.ErrDriverBusy
		}
		return err
	}

	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
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

	exec := txmanager.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	var t model.Trip
	err = row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &t.Status, &t.StartedAt,
		&t.FinishedAt, &t.LastPositionAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrTripNotFound
		}
		return nil, err
	}

	return &t, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error) {
	query, args, err := r.sb.Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Where(sq.Eq{"id": id, "status": "active"}).
		Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at, last_position_at").
		ToSql()
	if err != nil {
		return nil, err
	}

	exec := txmanager.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	var t model.Trip
	err = row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &t.Status, &t.StartedAt,
		&t.FinishedAt, &t.LastPositionAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, getErr := r.GetByID(ctx, id)
			if getErr != nil {
				return nil, getErr
			}
			if existing.Status == "completed" {
				return nil, model.ErrTripCompleted
			}
			return nil, model.ErrTripNotFound
		}
		return nil, err
	}

	return &t, nil
}
