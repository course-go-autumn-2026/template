package idempotency

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/toooriii/tori/internal/domain/trip"
)

const pgUniqueViolation = "23505"

func (r *Repo) Save(ctx context.Context, key uuid.UUID, hash string, tripID uuid.UUID) error {
	q, args, err := psql.Insert("idempotency_keys").
		Columns("key", "request_hash", "trip_id").
		Values(key, hash, tripID).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert idempotency key: %w", err)
	}
	if _, err := r.q.Querier(ctx).Exec(ctx, q, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return trip.ErrKeyAlreadyExists
		}
		return fmt.Errorf("insert idempotency key: %w", err)
	}
	return nil
}
