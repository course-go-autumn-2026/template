package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	domaintrip "github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/infra/postgres/transaction"
)

func (t *TripRepository) CreateTrip(ctx context.Context, trip domaintrip.Trip) error {
	executor := transaction.ExecutorFromContext(ctx, t.pool)

	query, args, err := sq.
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
			trip.ID,
			trip.UserID,
			trip.DriverID,
			trip.StartPoint.Latitude,
			trip.StartPoint.Longitude,
			trip.EndPoint.Latitude,
			trip.EndPoint.Longitude,
			trip.Price,
			trip.Status,
			trip.StartedAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	_, err = executor.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" &&
			pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
			return domaintrip.ErrDriverBusy
		}

		return fmt.Errorf("create trip: %w", err)
	}

	return nil
}

func (t *TripRepository) AddStatusHistory(ctx context.Context, tripId uuid.UUID, fromStatus *domaintrip.Status, toStatus domaintrip.Status, reason string) error {
	executor := transaction.ExecutorFromContext(ctx, t.pool)

	query, args, err := sq.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
		).
		Values(
			tripId,
			fromStatus,
			toStatus,
			reason,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add status history query: %w", err)
	}

	_, err = executor.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add status history: %w", err)
	}

	return nil
}

func (t *TripRepository) FinishTrip(ctx context.Context, tripId uuid.UUID, finishedAt time.Time) error {
	executor := transaction.ExecutorFromContext(ctx, t.pool)

	query, args, err := sq.
		Update("trips").
		Set("status", domaintrip.StatusCompleted).
		Set("finished_at", finishedAt).
		Set("updated_at", finishedAt).
		Where(sq.Eq{"id": tripId}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}

	_, err = executor.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}

	return nil
}
