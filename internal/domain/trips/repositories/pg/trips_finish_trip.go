package pg

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/makkssik/template/internal/domain/trips"
	"github.com/makkssik/template/internal/postgres"
	"github.com/samber/lo"
)

func (t *tripsRepository) FinishTrip(ctx context.Context, dto trips.FinishTripDTO) (*trips.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, t.queryTimeout)
	defer cancel()

	query, args, err := sq.Update("trips").
		Set("status", string(trips.StatusCompleted)).
		Set("finished_at", dto.FinishedAt).
		Set("updated_at", dto.FinishedAt).
		Where(sq.Eq{"id": dto.TripID, "status": string(trips.StatusActive)}).
		Suffix("RETURNING " + tripColumns).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build finish trip query: %w", err)
	}

	executor := postgres.ExecutorFromContext(ctx, t.pool)
	rows, err := executor.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}

	model, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Trip])
	if err == nil {
		return lo.ToPtr(dbToDomain(model)), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("scan finished trip: %w", err)
	}

	query, args, err = sq.Select("1").
		From("trips").
		Where(sq.Eq{"id": dto.TripID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build check trip query: %w", err)
	}
	var exists int
	err = executor.QueryRow(ctx, query, args...).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("finish trip: %w", trips.ErrTripNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("check trip after finish: %w", err)
	}
	return nil, fmt.Errorf("finish trip: %w", trips.ErrTripCompleted)
}
