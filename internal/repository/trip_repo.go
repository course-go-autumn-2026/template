package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/riiin11/labs/internal/txmanager"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrDriverBusy    = errors.New("driver busy")
	ErrTripCompleted = errors.New("trip completed")
)

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartLat   float64
	StartLon   float64
	EndLat     float64
	EndLon     float64
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type TripRepo struct {
	pool *pgxpool.Pool
}

func NewTripRepo(pool *pgxpool.Pool) *TripRepo {
	return &TripRepo{pool: pool}
}

func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (r *TripRepo) Insert(ctx context.Context, t Trip) error {
	exec := txmanager.FromContext(ctx, r.pool)

	q, args, err := psql.Insert("trips").
		Columns("id", "user_id", "driver_id",
			"start_latitude", "start_longitude", "end_latitude", "end_longitude",
			"price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID,
			t.StartLat, t.StartLon, t.EndLat, t.EndLon,
			t.Price, t.Status, t.StartedAt).
		ToSql()
	if err != nil {
		return err
	}
	if _, err := exec.Exec(ctx, q, args...); err != nil {
		if IsUniqueViolation(err) {
			return ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}

	qh, argsh, err := psql.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(t.ID, nil, "active", "trip created").
		ToSql()
	if err != nil {
		return err
	}
	if _, err := exec.Exec(ctx, qh, argsh...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}

func (r *TripRepo) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	exec := txmanager.FromContext(ctx, r.pool)
	q, args, err := psql.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude", "end_latitude", "end_longitude",
		"price", "status", "started_at", "finished_at").
		From("trips").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return Trip{}, err
	}
	var t Trip
	err = exec.QueryRow(ctx, q, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLat, &t.StartLon, &t.EndLat, &t.EndLon,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, ErrTripNotFound
	}
	return t, err
}

func (r *TripRepo) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	exec := txmanager.FromContext(ctx, r.pool)

	q, args, err := psql.Update("trips").
		Set("status", "completed").
		Set("finished_at", sq.Expr("now()")).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"status": "active"}}).
		ToSql()
	if err != nil {
		return Trip{}, err
	}
	tag, err := exec.Exec(ctx, q, args...)
	if err != nil {
		return Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	if tag.RowsAffected() == 0 {
		if _, err := r.Get(ctx, id); errors.Is(err, ErrTripNotFound) {
			return Trip{}, ErrTripNotFound
		}
		return Trip{}, ErrTripCompleted
	}
	return r.Get(ctx, id)
}