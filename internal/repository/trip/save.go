package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

func (r *Repository) Save(ctx context.Context, t trip.Trip) error {
	query, args, err := sq.Insert("trips").
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
			"last_position_at",
		).
		Values(
			t.ID,
			t.UserID,
			t.DriverID,
			t.StartPoint.Latitude,
			t.StartPoint.Longitude,
			t.EndPoint.Latitude,
			t.EndPoint.Longitude,
			t.Price,
			t.Status,
			t.StartedAt,
			t.FinishedAt,
			t.LastPositionAt,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}

	_, err = r.executor(ctx).Exec(ctx, query, args...)
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) &&
		pgErr.Code == "23505" &&
		pgErr.ConstraintName == "trips_one_active_per_driver_idx" {
		return fmt.Errorf("save trip: %w", errors.Join(trip.ErrDriverBusy, err))
	}

	return fmt.Errorf("save trip: %w", err)
}

func (r *Repository) Update(ctx context.Context, t trip.Trip) error {
	query, args, err := sq.
		Update("trips").
		Set("user_id", t.UserID).
		Set("driver_id", t.DriverID).
		Set("start_latitude", t.StartPoint.Latitude).
		Set("start_longitude", t.StartPoint.Longitude).
		Set("end_latitude", t.EndPoint.Latitude).
		Set("end_longitude", t.EndPoint.Longitude).
		Set("price", t.Price).
		Set("status", t.Status).
		Set("started_at", t.StartedAt).
		Set("finished_at", t.FinishedAt).
		Set("last_position_at", t.LastPositionAt).
		Set("updated_at", sq.Expr("now()")).
		Where(sq.Eq{"id": t.ID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update trip query: %w", err)
	}

	commandTag, err := r.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return trip.ErrTripNotFound
	}

	return nil
}

func (r *Repository) AddTransition(
	ctx context.Context,
	tripID uuid.UUID,
	fromStatus *trip.Status,
	toStatus trip.Status,
	reason *string,
) error {
	query, args, err := sq.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
		).
		Values(
			tripID,
			fromStatus,
			toStatus,
			reason,
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip status history query: %w", err)
	}

	_, err = r.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert trip status history: %w", err)
	}

	return nil
}
