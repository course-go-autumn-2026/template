package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Yolanesso/avito-labs/internal/model"
	"github.com/Yolanesso/avito-labs/internal/txmanager"
)

func (r *TripRepository) Create(
	ctx context.Context,
	trip model.Trip,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	builder := squirrel.Insert("trips")

	builder = builder.Columns(
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
	)

	builder = builder.Values(
		trip.ID,
		trip.UserID,
		trip.DriverID,
		trip.StartPoint.Latitude,
		trip.StartPoint.Longitude,
		trip.EndPoint.Latitude,
		trip.EndPoint.Longitude,
		trip.Price,
		string(trip.Status),
		trip.StartedAt,
		trip.FinishedAt,
		trip.CreatedAt,
		trip.UpdatedAt,
	)

	builder = builder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build create trip query: %w", err)
	}

	executor := txmanager.ExecutorFromContext(ctx, r.pool)

	_, err = executor.Exec(ctx, query, args...)
	if err == nil {
		return nil
	}

	var postgresError *pgconn.PgError

	if errors.As(err, &postgresError) {
		if postgresError.Code == "23505" {
			return model.ErrDriverBusy
		}
	}

	return fmt.Errorf("create trip: %w", err)
}
