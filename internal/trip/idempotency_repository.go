package trip

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/database"
)

type IdempotencyRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewIdempotencyRepository(pool *pgxpool.Pool, timeout time.Duration) *IdempotencyRepository {
	return &IdempotencyRepository{pool: pool, timeout: timeout}
}

func (r *IdempotencyRepository) Claim(ctx context.Context, key uuid.UUID, hash []byte, expiresAt time.Time) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Insert("trip_idempotency").
		Columns("idempotency_key", "request_hash", "expires_at").Values(key, hash, expiresAt).
		Suffix(`ON CONFLICT (idempotency_key) DO UPDATE
			SET request_hash = EXCLUDED.request_hash, expires_at = EXCLUDED.expires_at,
			    trip_id = NULL, response = NULL, created_at = now()
			WHERE trip_idempotency.expires_at <= now()
			RETURNING idempotency_key`).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return false, fmt.Errorf("build claim idempotency query: %w", err)
	}
	var claimed uuid.UUID
	err = database.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		// ON CONFLICT блокирует строку до конца транзакции даже при ложном WHERE.
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim idempotency key: %w", err)
	}
	return true, nil
}

func (r *IdempotencyRepository) Replay(ctx context.Context, key uuid.UUID, hash []byte) (api.Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Select("request_hash", "response").From("trip_idempotency").
		Where(sq.Eq{"idempotency_key": key}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return api.Trip{}, fmt.Errorf("build replay idempotency query: %w", err)
	}
	var storedHash, response []byte
	if err := database.ExecutorFromContext(ctx, r.pool).QueryRow(ctx, query, args...).Scan(&storedHash, &response); err != nil {
		return api.Trip{}, fmt.Errorf("read idempotency response: %w", err)
	}
	if !bytes.Equal(storedHash, hash) {
		return api.Trip{}, ErrIdempotencyConflict
	}
	var value api.Trip
	if err := json.Unmarshal(response, &value); err != nil {
		return api.Trip{}, fmt.Errorf("decode idempotency response: %w", err)
	}
	return value, nil
}

func (r *IdempotencyRepository) Save(ctx context.Context, key uuid.UUID, value api.Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	response, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode idempotency response: %w", err)
	}
	query, args, err := sq.Update("trip_idempotency").Set("trip_id", value.Id).Set("response", response).
		Where(sq.Eq{"idempotency_key": key}).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build save idempotency query: %w", err)
	}
	if _, err := database.ExecutorFromContext(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("save idempotency response: %w", err)
	}
	return nil
}

func (r *IdempotencyRepository) Prune(ctx context.Context, currentKey uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	expired := sq.Select("idempotency_key").From("trip_idempotency").
		Where("expires_at <= now()").Where(sq.NotEq{"idempotency_key": currentKey}).
		Limit(100).Suffix("FOR UPDATE SKIP LOCKED")
	query, args, err := sq.Delete("trip_idempotency").Where(sq.Expr("idempotency_key IN (?)", expired)).
		PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build prune idempotency query: %w", err)
	}
	if _, err := database.ExecutorFromContext(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("prune expired idempotency keys: %w", err)
	}
	return nil
}
