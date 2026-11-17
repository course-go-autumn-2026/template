package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/infra/postgres/transaction"
)

func (t *TripRepository) GetTrip(ctx context.Context, tripId uuid.UUID) (domaintrip.Trip, error) {
	return t.getTrip(ctx, tripId, false)
}

func (t *TripRepository) GetTripForUpdate(ctx context.Context, tripId uuid.UUID) (domaintrip.Trip, error) {
	return t.getTrip(ctx, tripId, true)
}

func (t *TripRepository) getTrip(ctx context.Context, tripId uuid.UUID, forUpdate bool) (domaintrip.Trip, error) {
	executor := transaction.ExecutorFromContext(ctx, t.pool)

	builder := sq.Select(
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
		From("trips").
		Where(sq.Eq{"id": tripId}).
		PlaceholderFormat(sq.Dollar)

	if forUpdate {
		builder = builder.Suffix("FOR UPDATE")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}

	var trip domaintrip.Trip

	err = executor.QueryRow(ctx, query, args...).Scan(
		&trip.ID,
		&trip.UserID,
		&trip.DriverID,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.Price,
		&trip.Status,
		&trip.StartedAt,
		&trip.FinishedAt,
		&trip.LastPositionAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domaintrip.Trip{}, domaintrip.ErrTripNotFound
		}

		return domaintrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return trip, nil
}
