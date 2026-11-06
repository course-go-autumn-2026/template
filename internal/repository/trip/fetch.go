package trip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

var tripColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at",
}

func (r *TripRepository) Get(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	q, args, err := psql.Select(tripColumns...).From("trips").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build select trip: %w", err)
	}
	t, err := scanTrip(r.q.Querier(ctx).QueryRow(ctx, q, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, trip.ErrTripNotFound
	}
	if err != nil {
		return model.Trip{}, fmt.Errorf("select trip: %w", err)
	}
	return t, nil
}

func (r *TripRepository) Finish(ctx context.Context, id uuid.UUID, at time.Time) (model.Trip, error) {
	q, args, err := psql.Update("trips").
		Set("status", string(trip.StatusCompleted)).
		Set("finished_at", at).
		Set("updated_at", at).
		Where(sq.Eq{"id": id, "status": string(trip.StatusActive)}).
		Suffix("RETURNING " + strings.Join(tripColumns, ", ")).
		ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build update trip: %w", err)
	}

	t, err := scanTrip(r.q.Querier(ctx).QueryRow(ctx, q, args...))
	if err == nil {
		return t, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, fmt.Errorf("update trip: %w", err)
	}

	if _, getErr := r.Get(ctx, id); getErr != nil {
		return model.Trip{}, getErr
	}
	return model.Trip{}, trip.ErrTripCompleted
}
