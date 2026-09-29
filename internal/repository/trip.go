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

	"trip-service/internal/pkg/txmanager"
)

const pgErrUniqueViolation = "23505"

type TripRepository struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *TripRepository) getExecutor(ctx context.Context) txmanager.DBTX {
	return txmanager.GetDBTX(ctx, r.pool)
}

func (r *TripRepository) Create(ctx context.Context, trip Trip) error {
	query, args, err := r.builder.
		Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
		).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartLatitude, trip.StartLongitude,
			trip.EndLatitude, trip.EndLongitude,
			trip.Price, trip.Status, trip.StartedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	exec := r.getExecutor(ctx)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgErrUniqueViolation {
			return ErrDriverBusy
		}
		return fmt.Errorf("exec create trip query: %w", err)
	}

	return nil
}

func (r *TripRepository) AddStatusHistory(ctx context.Context, tripID uuid.UUID, status string, changedAt time.Time) error {
	query, args, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "status", "changed_at").
		Values(tripID, status, changedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add status history query: %w", err)
	}

	exec := r.getExecutor(ctx)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec add status history query: %w", err)
	}
	return nil
}

func (r *TripRepository) GetByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	query, args, err := r.builder.
		Select(
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
		return nil, fmt.Errorf("build get trip query: %w", err)
	}

	exec := r.getExecutor(ctx)
	row := exec.QueryRow(ctx, query, args...)

	var trip Trip
	err = row.Scan(
		&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartLatitude, &trip.StartLongitude,
		&trip.EndLatitude, &trip.EndLongitude,
		&trip.Price, &trip.Status, &trip.StartedAt,
		&trip.FinishedAt, &trip.LastPositionAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip row: %w", err)
	}

	return &trip, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*Trip, error) {
	query, args, err := r.builder.
		Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": id, "status": "active"}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build finish trip query: %w", err)
	}

	exec := r.getExecutor(ctx)
	cmdTag, err := exec.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec finish trip query: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		trip, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if trip.Status == "completed" {
			return nil, ErrTripCompleted
		}
		return nil, ErrTripNotFound
	}

	return r.GetByID(ctx, id)
}
