package trip

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func (r *Repository) scan(ctx context.Context, query string, args ...any) (model.Trip, error) {
	var t model.Trip

	err := postgres.Executor(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&t.ID, &t.UserID, &t.DriverID, &t.StartPoint.Latitude, &t.StartPoint.Longitude, &t.EndPoint.Latitude, &t.EndPoint.Longitude, &t.Price, &t.Status, &t.StartedAt, &t.FinishedAt, &t.LastPositionAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Trip{}, trip.ErrTripNotFound
	}

	if err != nil {
		return model.Trip{}, fmt.Errorf("query trip: %w", err)
	}

	return t, nil
}
