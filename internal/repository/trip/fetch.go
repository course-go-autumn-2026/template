package trip

import (
	"context"
	"errors"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/infra/postgres/transaction"
	"github.com/XrystikLK/template/internal/model"
)

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.sq.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude",
		"end_latitude", "end_longitude",
		"price", "status", "started_at", "finished_at",
		"created_at", "updated_at",
	).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	var exec executor = transaction.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	t, err := scanTrip(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainTrip.ErrTripNotFound
	}
	if err != nil {
		return nil, err
	}

	return t, nil
}
