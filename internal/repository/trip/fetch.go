package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

func (r *Repository) GetTrip(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	query, args, err := sq.
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
			"last_position_at",
		).
		From("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("build select trip query: %w", err)
	}

	row := r.executor(ctx).QueryRow(ctx, query, args...)
	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, trip.ErrTripNotFound
	}
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return t, nil
}

func (r *Repository) GetTripForUpdate(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	query, args, err := sq.
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
			"last_position_at",
		).
		From("trips").
		Where(sq.Eq{"id": id}).
		Suffix("FOR UPDATE").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return trip.Trip{}, fmt.Errorf("build select trip query: %w", err)
	}

	row := r.executor(ctx).QueryRow(ctx, query, args...)
	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return trip.Trip{}, trip.ErrTripNotFound
	}
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return t, nil
}

func (r *Repository) GetTripsSurvey(ctx context.Context,
	positionStaleAfter time.Duration,
	batchSize int,
	lastTime time.Time,
	lastID uuid.UUID,
) ([]trip.Trip, error) {
	query := "SELECT id, user_id, driver_id, " +
		"start_latitude, start_longitude, " +
		"end_latitude, end_longitude, price, " +
		"status, started_at, finished_at, last_position_at " +
		"FROM trips " +
		"WHERE status = 'active' " +
		"AND (CASE WHEN last_position_at IS NOT NULL THEN last_position_at ELSE started_at END) <= now() - make_interval(secs => $1) " +
		"AND ((CASE WHEN last_position_at IS NOT NULL THEN last_position_at ELSE started_at END), id) > ($3, $4) " +
		"ORDER BY (CASE WHEN last_position_at IS NOT NULL THEN last_position_at ELSE started_at END) ASC, id ASC " +
		"LIMIT $2;"

	rows, err := r.executor(ctx).Query(ctx, query, positionStaleAfter.Seconds(), batchSize, lastTime, lastID)
	if err != nil {
		return nil, fmt.Errorf("query rows: %w", err)
	}
	defer rows.Close()
	var trips []trip.Trip
	for rows.Next() {
		var t trip.Trip
		t, err = scanTrip(rows)
		if err != nil {
			return []trip.Trip{}, fmt.Errorf("scan trip: %w", err)
		}
		trips = append(trips, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("problem wit rows: %w", err)
	}
	return trips, nil
}
