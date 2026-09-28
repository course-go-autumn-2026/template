package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/poiuyttq/tripgo/internal/model"
	"github.com/poiuyttq/tripgo/internal/txmanager"
)

var ErrDriverHasActiveTrip = errors.New("driver already has active trip")

type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

func (r *TripRepository) exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if tx := txmanager.TxFromContext(ctx); tx != nil {
		return tx.Exec(ctx, query, args...)
	}
	return r.pool.Exec(ctx, query, args...)
}

func (r *TripRepository) queryRow(ctx context.Context, query string, args ...any) pgx.Row {
	if tx := txmanager.TxFromContext(ctx); tx != nil {
		return tx.QueryRow(ctx, query, args...)
	}
	return r.pool.QueryRow(ctx, query, args...)
}

func (r *TripRepository) Create(ctx context.Context, trip model.Trip) error {
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
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
			"finished_at",
			"created_at",
			"updated_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartLat,
			trip.StartLng,
			trip.EndLat,
			trip.EndLng,
			trip.Price,
			trip.Status,
			trip.StartedAt,
			trip.FinishedAt,
			trip.CreatedAt,
			trip.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	if _, err := r.exec(ctx, query, args...); err != nil {
		if isUniqueViolation(err) {
			return ErrDriverHasActiveTrip
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}

func (r *TripRepository) GetByID(ctx context.Context, id string) (model.Trip, error) {
	var trip model.Trip
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
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
			"created_at",
			"updated_at",
		).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	err = r.queryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartLat,
		&trip.StartLng,
		&trip.EndLat,
		&trip.EndLng,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, fmt.Errorf("trip not found: %w", err)
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}

func (r *TripRepository) GetActiveByDriverID(ctx context.Context, driverID string) (model.Trip, bool, error) {
	var trip model.Trip
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
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
			"created_at",
			"updated_at",
		).
		From("trips").
		Where(squirrel.Eq{"driver_id": driverID}).
		Where(squirrel.Eq{"status": "active"}).
		Limit(1).
		ToSql()
	if err != nil {
		return model.Trip{}, false, fmt.Errorf("build active trip query: %w", err)
	}

	err = r.queryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartLat,
		&trip.StartLng,
		&trip.EndLat,
		&trip.EndLng,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.CreatedAt,
		&trip.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, false, nil
	}
	if err != nil {
		return model.Trip{}, false, fmt.Errorf("get active trip by driver: %w", err)
	}

	return trip, true, nil
}

func (r *TripRepository) Finish(ctx context.Context, tripID string, finishedAt time.Time) error {
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}

	result, err := r.exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trip not found")
	}
	return nil
}

func (r *TripRepository) FinishTx(ctx context.Context, tx pgx.Tx, tripID string, finishedAt time.Time) error {
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": tripID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}
	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip in tx: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("trip not found")
	}
	return nil
}

func (r *TripRepository) CreateTx(ctx context.Context, tx pgx.Tx, trip model.Trip) error {
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
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
			"finished_at",
			"created_at",
			"updated_at",
		).
		Values(
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartLat,
			trip.StartLng,
			trip.EndLat,
			trip.EndLng,
			trip.Price,
			trip.Status,
			trip.StartedAt,
			trip.FinishedAt,
			trip.CreatedAt,
			trip.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query in tx: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert trip in tx: %w", err)
	}
	return nil
}
