package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	domaintrip "github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/infra/postgres/transaction"
)

func (t *TripRepository) SavePosition(ctx context.Context, id uuid.UUID, data domaintrip.PositionData) (domaintrip.Position, error) {
	executor := transaction.ExecutorFromContext(ctx, t.pool)
	query, args, err := sq.Insert("trip_positions").Columns("trip_id", "latitude", "longitude", "recorded_at").
		Values(id, data.Coordinates.Latitude, data.Coordinates.Longitude, data.RecordedAt).
		Suffix("RETURNING id").PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("build position insert: %w", err)
	}

	position := domaintrip.Position{TripID: id, PositionData: data}

	if err := executor.QueryRow(ctx, query, args...).Scan(&position.ID); err != nil {
		return domaintrip.Position{}, fmt.Errorf("insert position: %w", err)
	}

	_, err = executor.Exec(
		ctx,
		`UPDATE trips SET last_position_at = GREATEST(last_position_at, $2::timestamptz), updated_at = now() WHERE id = $1`,
		id,
		data.RecordedAt,
	)
	if err != nil {
		return domaintrip.Position{}, fmt.Errorf("update last position: %w", err)
	}

	return position, nil
}

func (t *TripRepository) ListPositions(ctx context.Context, id uuid.UUID) ([]domaintrip.Position, error) {
	executor := transaction.ExecutorFromContext(ctx, t.pool)
	rows, err := executor.Query(
		ctx,
		`SELECT id, trip_id, latitude, longitude, recorded_at FROM trip_positions WHERE trip_id = $1 ORDER BY recorded_at, id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("list positions: %w", err)
	}

	defer rows.Close()

	positions := make([]domaintrip.Position, 0)

	for rows.Next() {
		var p domaintrip.Position

		if err := rows.Scan(&p.ID, &p.TripID, &p.Coordinates.Latitude, &p.Coordinates.Longitude, &p.RecordedAt); err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}

		positions = append(positions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read positions: %w", err)
	}

	return positions, nil
}

func (t *TripRepository) FindStaleTrips(ctx context.Context, staleAfter time.Duration) ([]domaintrip.StaleTrip, error) {
	rows, err := t.pool.Query(ctx, `SELECT id, driver_id, COALESCE(last_position_at, started_at) FROM trips
 WHERE status = 'active' AND COALESCE(last_position_at, started_at) < now() - $1::double precision * interval '1 second'
 ORDER BY COALESCE(last_position_at, started_at), id`, staleAfter.Seconds())
	if err != nil {
		return nil, fmt.Errorf("find stale trips: %w", err)
	}

	defer rows.Close()

	trips := make([]domaintrip.StaleTrip, 0)

	for rows.Next() {
		var entity domaintrip.StaleTrip

		if err := rows.Scan(&entity.ID, &entity.DriverID, &entity.LastPositionAt); err != nil {
			return nil, fmt.Errorf("scan stale trip: %w", err)
		}

		trips = append(trips, entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read stale trips: %w", err)
	}

	return trips, nil
}

func (t *TripRepository) NeedsPosition(ctx context.Context, id uuid.UUID, staleAfter time.Duration) (bool, error) {
	var needed bool

	err := t.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM trips WHERE id = $1 AND status = 'active'
 AND COALESCE(last_position_at, started_at) < now() - $2::double precision * interval '1 second')`, id, staleAfter.Seconds()).Scan(&needed)
	if err != nil {
		return false, fmt.Errorf("check stale trip: %w", err)
	}

	return needed, nil
}
