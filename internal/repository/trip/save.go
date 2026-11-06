package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

const (
	pgUniqueViolation = "23505"
	activeDriverIndex = "trips_driver_active_uidx"
)

func (r *TripRepository) Save(ctx context.Context, t model.Trip) error {
	q, args, err := psql.Insert("trips").
		Columns("id", "user_id", "driver_id",
			"start_latitude", "start_longitude", "end_latitude", "end_longitude",
			"price", "status", "started_at").
		Values(t.ID, t.UserID, t.DriverID,
			t.Start.Latitude, t.Start.Longitude, t.End.Latitude, t.End.Longitude,
			t.Price, string(t.Status), t.StartedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert trip: %w", err)
	}
	if _, err := r.q.Querier(ctx).Exec(ctx, q, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == activeDriverIndex {
			return trip.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}
	return nil
}

func (r *TripRepository) AddHistory(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error {
	q, args, err := psql.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, from, to, reason).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert history: %w", err)
	}
	if _, err := r.q.Querier(ctx).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("insert history: %w", err)
	}
	return nil
}
