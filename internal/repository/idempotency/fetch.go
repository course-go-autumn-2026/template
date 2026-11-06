package idempotency

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

func (r *Repo) Get(ctx context.Context, key uuid.UUID) (model.IdempotencyRecord, error) {
	q, args, err := psql.Select("key", "request_hash", "trip_id").
		From("idempotency_keys").
		Where(sq.Eq{"key": key}).
		ToSql()
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("build select idempotency key: %w", err)
	}

	var rec model.IdempotencyRecord
	err = r.q.Querier(ctx).QueryRow(ctx, q, args...).Scan(&rec.Key, &rec.RequestHash, &rec.TripID)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.IdempotencyRecord{}, trip.ErrTripNotFound
	}
	if err != nil {
		return model.IdempotencyRecord{}, fmt.Errorf("select idempotency key: %w", err)
	}
	return rec, nil
}
