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

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/infra/postgres/transaction"
)

const tripColumns = "id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at"

func (r *Repository) Create(ctx context.Context, value domaintrip.Trip) (domaintrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_latitude", "start_longitude", "end_latitude", "end_longitude", "price", "status", "started_at").
		Values(value.Id, value.UserId, value.DriverId, value.StartPoint.Latitude, value.StartPoint.Longitude,
			value.EndPoint.Latitude, value.EndPoint.Longitude, value.Price, domaintrip.StatusActive, value.StartedAt).
		Suffix("RETURNING " + tripColumns).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build create trip query: %w", err)
	}
	created, err := scanTrip(transaction.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_active_driver_uniq" {
			return domaintrip.Trip{}, domaintrip.ErrDriverBusy
		}
		return domaintrip.Trip{}, fmt.Errorf("create trip: %w", err)
	}
	return created, nil
}

func (r *Repository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domaintrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Update("trips").
		Set("status", domaintrip.StatusCompleted).Set("finished_at", finishedAt).Set("updated_at", finishedAt).
		Where(sq.Eq{"id": id, "status": domaintrip.StatusActive}).
		Suffix("RETURNING " + tripColumns).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build finish trip query: %w", err)
	}
	value, err := scanTrip(transaction.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := r.Get(ctx, id); err != nil {
			return domaintrip.Trip{}, err
		}
		return domaintrip.Trip{}, domaintrip.ErrTripCompleted
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("finish trip: %w", err)
	}
	return value, nil
}
