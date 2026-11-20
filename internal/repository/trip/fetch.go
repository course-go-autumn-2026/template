package trip

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

var selectColumns = []string{
	"id", "user_id", "driver_id",
	"start_latitude", "start_longitude", "end_latitude", "end_longitude",
	"price", "status", "started_at", "finished_at", "last_position_at",
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	q, args, err := sql.Select(selectColumns...).From("trips").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build get trip: %w", err)
	}

	return r.scan(ctx, q, args...)
}

// GetForUpdate must be called inside TxManager.Do so the lock is actually held.
func (r *Repository) GetForUpdate(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	q, args, err := sql.Select(selectColumns...).From("trips").Where(squirrel.Eq{"id": id}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return model.Trip{}, fmt.Errorf("build get trip for update: %w", err)
	}

	return r.scan(ctx, q, args...)
}

// ClaimIdempotencyKey reserves a client key in the current transaction.
func (r *Repository) ClaimIdempotencyKey(ctx context.Context, key uuid.UUID, hash []byte) (claimed bool, existingHash []byte, existingTripID uuid.UUID, err error) {
	// Opportunistic cleanup: records older than 24h, no separate worker needed.
	cleanup, cleanupArgs, buildErr := sql.Delete("idempotency_keys").Where("created_at < now() - interval '24 hours'").ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency cleanup: %w", buildErr)
	}

	exec := postgres.Executor(ctx, r.pool)
	if _, err = exec.Exec(ctx, cleanup, cleanupArgs...); err != nil {
		return false, nil, uuid.Nil, fmt.Errorf("delete expired idempotency keys: %w", err)
	}

	q, args, buildErr := sql.Insert("idempotency_keys").Columns("key", "request_hash", "trip_id").Values(key, hash, nil).Suffix("ON CONFLICT (key) DO NOTHING RETURNING key").ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency claim: %w", buildErr)
	}

	var returned uuid.UUID

	err = exec.QueryRow(ctx, q, args...).Scan(&returned)
	if err == nil {
		return true, nil, uuid.Nil, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return false, nil, uuid.Nil, fmt.Errorf("claim idempotency key: %w", err)
	}

	q, args, buildErr = sql.Select("request_hash", "trip_id").From("idempotency_keys").Where(squirrel.Eq{"key": key}).ToSql()
	if buildErr != nil {
		return false, nil, uuid.Nil, fmt.Errorf("build idempotency lookup: %w", buildErr)
	}

	if err = exec.QueryRow(ctx, q, args...).Scan(&existingHash, &existingTripID); err != nil {
		return false, nil, uuid.Nil, fmt.Errorf("lookup idempotency key: %w", err)
	}

	return false, existingHash, existingTripID, nil
}

func (r *Repository) CompleteIdempotencyKey(ctx context.Context, key, tripID uuid.UUID) error {
	q, args, err := sql.Update("idempotency_keys").Set("trip_id", tripID).Where(squirrel.Eq{"key": key}).ToSql()
	if err != nil {
		return fmt.Errorf("build idempotency completion: %w", err)
	}

	if _, err = postgres.Executor(ctx, r.pool).Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("complete idempotency key: %w", err)
	}

	return nil
}

// AddStatusHistory records a status transition.
func (r *Repository) AddStatusHistory(ctx context.Context, id uuid.UUID, from *string, to, reason string, changedAt time.Time) error {
	q, args, err := sql.Insert("trip_status_history").Columns("trip_id", "from_status", "to_status", "reason", "changed_at").Values(id, from, to, reason, changedAt).ToSql()
	if err != nil {
		return fmt.Errorf("build status history: %w", err)
	}

	_, err = postgres.Executor(ctx, r.pool).Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("insert status history: %w", err)
	}

	return nil
}
