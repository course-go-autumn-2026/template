package position

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

func (r *Repository) UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, lastPositionAt time.Time) error {
	query := "UPDATE trips " +
		"SET last_position_at = CASE " +
		"WHEN last_position_at IS NULL OR $2 > last_position_at THEN $2 " +
		"ELSE last_position_at END, " +
		"updated_at = now() " +
		"WHERE id = $1"
	commandTag, err := r.executor(ctx).Exec(ctx, query, tripID, lastPositionAt)
	if err != nil {
		return fmt.Errorf("update trip: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return trip.ErrTripNotFound
	}

	return nil
}

func (r *Repository) AddPosition(ctx context.Context, position *trip.Position) (*trip.Position, error) {
	query, args, err := sq.
		Insert("trip_positions").
		Columns("trip_id",
			"latitude",
			"longitude",
			"recorded_at",
		).
		Values(position.TripID,
			position.Coordinates.Latitude,
			position.Coordinates.Longitude,
			position.RecordedAt).
		Suffix("RETURNING id, trip_id, latitude, longitude, recorded_at, created_at").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build insert position query: %w", err)
	}
	row := r.executor(ctx).QueryRow(ctx, query, args...)
	newPosition, err := scanPosition(row)
	if err != nil {
		return nil, fmt.Errorf("insert position: %w", err)
	}

	return &newPosition, nil
}
