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
	"github.com/razrabbesezdav-art/template/internal/transaction"
)

var (
	ErrTripNotFound  = errors.New("trip not found")
	ErrTripCompleted = errors.New("trip already completed")
	ErrDriverBusy    = errors.New("driver already has an active trip")
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

type TripData struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	StartLat float64
	StartLon float64
	EndLat   float64
	EndLon   float64
	Price    int64
}

type Querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Trips struct {
	db           Querier
	builder      sq.StatementBuilderType
	queryTimeout time.Duration
}

func NewTrips(db Querier, queryTimeout time.Duration) *Trips {
	return &Trips{
		db:           db,
		builder:      sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		queryTimeout: queryTimeout,
	}
}

func (r *Trips) queryer(ctx context.Context) Querier {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}
	return r.db
}

func (r *Trips) Create(ctx context.Context, data TripData) (Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	id := uuid.New()
	startedAt := time.Now().UTC()

	query, args, err := r.builder.
		Insert("trips").
		Columns(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
		).
		Values(
			id,
			data.UserID,
			data.DriverID,
			data.StartLat,
			data.StartLon,
			data.EndLat,
			data.EndLon,
			data.Price,
			"active",
			startedAt,
		).
		Suffix(`RETURNING id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at`).
		ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf("build create trip query: %w", err)
	}

	trip, err := scanTrip(r.queryer(ctx).QueryRow(ctx, query, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return Trip{}, ErrDriverBusy
		}
		return Trip{}, fmt.Errorf("create trip: %w", err)
	}

	return trip, nil
}

func (r *Trips) Get(ctx context.Context, id uuid.UUID) (Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.builder.
		Select(
			"id",
			"user_id",
			"driver_id",
			"start_latitude",
			"start_longitude",
			"end_latitude",
			"end_longitude",
			"price",
			"status",
			"started_at",
			"finished_at",
		).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	trip, err := scanTrip(r.queryer(ctx).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, ErrTripNotFound
	}
	if err != nil {
		return Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (r *Trips) Finish(ctx context.Context, id uuid.UUID) (Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.builder.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", sq.Expr("now()")).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"status": "active"}).
		Suffix(`RETURNING id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at`).
		ToSql()
	if err != nil {
		return Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}

	trip, err := scanTrip(r.queryer(ctx).QueryRow(ctx, query, args...))
	if err == nil {
		return trip, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Trip{}, fmt.Errorf("finish trip: %w", err)
	}

	existing, getErr := r.Get(ctx, id)
	if errors.Is(getErr, ErrTripNotFound) {
		return Trip{}, ErrTripNotFound
	}
	if getErr != nil {
		return Trip{}, getErr
	}
	if existing.Status == "completed" {
		return Trip{}, ErrTripCompleted
	}

	return Trip{}, fmt.Errorf("finish trip: unexpected state")
}

func scanTrip(row pgx.Row) (Trip, error) {
	var trip Trip
	err := row.Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartLat,
		&trip.StartLon,
		&trip.EndLat,
		&trip.EndLon,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
	)
	return trip, err
}

var _ Querier = (*pgxpool.Pool)(nil)