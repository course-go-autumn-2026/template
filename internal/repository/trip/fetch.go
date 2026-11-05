package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
)

var tripColumns = []string{
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
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	exec := r.getExecutor(ctx)

	query, args, err := r.builder.
		Select(tripColumns...).
		From("trips").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select trip query: %w", err)
	}

	var d dbTrip
	row := exec.QueryRow(ctx, query, args...)
	err = row.Scan(
		&d.ID,
		&d.UserID,
		&d.DriverID,
		&d.StartLatitude,
		&d.StartLongitude,
		&d.EndLatitude,
		&d.EndLongitude,
		&d.Price,
		&d.Status,
		&d.StartedAt,
		&d.FinishedAt,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, trip.ErrTripNotFound
		}
		return nil, fmt.Errorf("scan trip: %w", err)
	}

	return d.toModel(), nil
}
