package trip

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/infra/postgres/transaction"
)

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Select(tripColumns).From("trips").Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("build get trip query: %w", err)
	}
	value, err := scanTrip(transaction.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return domaintrip.Trip{}, domaintrip.ErrTripNotFound
	}
	if err != nil {
		return domaintrip.Trip{}, fmt.Errorf("get trip: %w", err)
	}
	return value, nil
}
