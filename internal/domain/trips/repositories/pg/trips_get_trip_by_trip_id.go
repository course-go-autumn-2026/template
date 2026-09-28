package pg

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/makkssik/template/internal/domain/trips"
	"github.com/makkssik/template/internal/postgres"
	"github.com/samber/lo"
)

func (t *tripsRepository) GetTripByTripId(ctx context.Context, tripID uuid.UUID) (*trips.Trip, error) {
	query, args, err := sq.Select(tripColumns).
		From("trips").
		Where(sq.Eq{"id": tripID}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip query: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, t.queryTimeout)
	defer cancel()
	rows, err := postgres.ExecutorFromContext(ctx, t.pool).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}

	model, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Trip])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get trip: %w", trips.ErrTripNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("scan trip: %w", err)
	}
	return lo.ToPtr(dbToDomain(model)), nil
}
