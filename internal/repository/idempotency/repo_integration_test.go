//go:build integration

package idempotency_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/toooriii/tori/internal/config"
	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/infra/postgres"
	"github.com/toooriii/tori/internal/infra/postgres/transaction"
	repo "github.com/toooriii/tori/internal/repository/idempotency"
)

func TestIdempotencyRepository_SaveAndConflict(t *testing.T) {
	cfg, _ := config.Load()
	ctx := context.Background()
	pool, _ := postgres.NewPool(ctx, cfg.Database)
	defer pool.Close()

	_, _ = pool.Exec(ctx, "TRUNCATE idempotency_keys, trips RESTART IDENTITY CASCADE")

	tx := transaction.New(pool, pgx.ReadCommitted)
	r := repo.New(tx)

	key := uuid.New()
	tripID := uuid.New()
	_, _ = pool.Exec(ctx, "INSERT INTO trips (id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at) VALUES ($1, $2, $3, 1, 1, 2, 2, 100, 'active', now())", tripID, uuid.New(), uuid.New())

	if err := r.Save(ctx, key, "hash1", tripID); err != nil {
		t.Fatalf("save: %v", err)
	}

	err := r.Save(ctx, key, "hash2", tripID)
	if !errors.Is(err, trip.ErrKeyAlreadyExists) {
		t.Fatalf("err = %v, want ErrKeyAlreadyExists", err)
	}
}
