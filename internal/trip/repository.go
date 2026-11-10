package trip

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
	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/database"
)

const tripColumns = "id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at"

type Repository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewRepository(pool *pgxpool.Pool, timeout time.Duration) *Repository {
	return &Repository{pool: pool, timeout: timeout}
}

func (r *Repository) Create(ctx context.Context, value api.Trip) (api.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(value.Id, value.UserId, value.DriverId, value.StartPoint.Latitude, value.StartPoint.Longitude,
			value.EndPoint.Latitude, value.EndPoint.Longitude, value.Price, api.Active, value.StartedAt).
		Suffix("RETURNING " + tripColumns).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return api.Trip{}, fmt.Errorf("build create trip query: %w", err)
	}
	created, err := scanTrip(database.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_active_driver_uniq" {
			return api.Trip{}, ErrDriverBusy
		}
		return api.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	return created, nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (api.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Select(tripColumns).From("trips").Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return api.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}
	value, err := scanTrip(database.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return api.Trip{}, ErrTripNotFound
	}
	if err != nil {
		return api.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return value, nil
}

func (r *Repository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (api.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Update("trips").
		Set("status", api.Completed).Set("finished_at", finishedAt).Set("updated_at", finishedAt).
		Where(sq.Eq{"id": id, "status": api.Active}).
		Suffix("RETURNING " + tripColumns).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return api.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}
	value, err := scanTrip(database.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		// Условный UPDATE не различает отсутствующую и уже завершённую поездки.
		if _, err := r.Get(ctx, id); err != nil {
			return api.Trip{}, err
		}
		return api.Trip{}, ErrTripCompleted
	}
	if err != nil {
		return api.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return value, nil
}

func scanTrip(row pgx.Row) (api.Trip, error) {
	var value api.Trip
	err := row.Scan(&value.Id, &value.UserId, &value.DriverId,
		&value.StartPoint.Latitude, &value.StartPoint.Longitude,
		&value.EndPoint.Latitude, &value.EndPoint.Longitude,
		&value.Price, &value.Status, &value.StartedAt, &value.FinishedAt)
	value.StartedAt = value.StartedAt.UTC()
	if value.FinishedAt != nil {
		finishedAt := value.FinishedAt.UTC()
		value.FinishedAt = &finishedAt
	}
	return value, err
}
