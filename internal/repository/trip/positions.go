package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/model"
)

func (r *RepositoryImpl) UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, recordedAt time.Time) error {
	exec := transaction.GetExecutor(ctx, r.pool)
	const query = `
		UPDATE trips
		SET last_position_at = CASE
			WHEN last_position_at IS NULL OR $1 > last_position_at THEN $1
			ELSE last_position_at
		END
		WHERE id = $2 AND status = 'active'
	`

	tag, err := exec.Exec(ctx, query, recordedAt, tripID)
	if err != nil {
		return fmt.Errorf("update last_position_at: %w", err)
	}

	if tag.RowsAffected() == 0 {
		t, getErr := r.GetByID(ctx, tripID)
		if getErr != nil {
			return getErr
		}
		if t.Status != model.StatusActive {
			return domaintrip.ErrTripCompleted
		}
		return domaintrip.ErrTripNotFound
	}

	return nil
}

func (r *RepositoryImpl) InsertPosition(ctx context.Context, pos *model.Position) (int64, error) {
	exec := transaction.GetExecutor(ctx, r.pool)
	query, args, err := r.sb.
		Insert("trip_positions").
		Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(pos.TripID, pos.Coordinates.Latitude, pos.Coordinates.Longitude, pos.RecordedAt).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build insert position query: %w", err)
	}

	var id int64
	if err := exec.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert position: %w", err)
	}

	return id, nil
}

func (r *RepositoryImpl) ListPositions(ctx context.Context, tripID uuid.UUID) ([]model.Position, error) {
	if _, err := r.GetByID(ctx, tripID); err != nil {
		return nil, err
	}

	exec := transaction.GetExecutor(ctx, r.pool)
	query, args, err := r.sb.
		Select("id", "trip_id", "latitude", "longitude", "recorded_at").
		From("trip_positions").
		Where(sq.Eq{"trip_id": tripID}).
		OrderBy("recorded_at ASC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list positions query: %w", err)
	}

	rows, err := exec.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query positions: %w", err)
	}
	defer rows.Close()

	positions := make([]model.Position, 0)
	for rows.Next() {
		var p model.Position
		if err := rows.Scan(&p.ID, &p.TripID, &p.Coordinates.Latitude, &p.Coordinates.Longitude, &p.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate positions: %w", err)
	}

	return positions, nil
}

func (r *RepositoryImpl) FindStaleActiveTrips(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error) {
	exec := transaction.GetExecutor(ctx, r.pool)
	const query = `
		SELECT id, driver_id
		FROM trips
		WHERE status = 'active'
		  AND COALESCE(last_position_at, started_at) < NOW() - ($1 * INTERVAL '1 microsecond')
		ORDER BY COALESCE(last_position_at, started_at) ASC
	`

	rows, err := exec.Query(ctx, query, staleAfter.Microseconds())
	if err != nil {
		return nil, fmt.Errorf("query stale trips: %w", err)
	}
	defer rows.Close()

	trips := make([]model.StaleTrip, 0)
	for rows.Next() {
		var t model.StaleTrip
		if err := rows.Scan(&t.ID, &t.DriverID); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}
		trips = append(trips, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stale trips: %w", err)
	}

	return trips, nil
}
