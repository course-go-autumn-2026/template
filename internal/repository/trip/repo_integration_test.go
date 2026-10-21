//go:build integration

package trip

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	"github.com/whxtelyy/template/internal/infra/postgres/transaction"
)

func TestCreateRollsBackAfterInvalidHistoryInsert(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	repo := NewTripRepository(pool, 5*time.Second)
	tx := transaction.NewManager(pool, pgx.ReadCommitted, 5*time.Second)
	id := uuid.New()
	trip := domain.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Status: domain.StatusActive}
	defer func() { _, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id=$1", id) }()
	err = tx.Do(ctx, func(txCtx context.Context) error {
		if _, err := repo.Create(txCtx, trip); err != nil {
			return err
		}
		return repo.AddStatusHistory(txCtx, id, nil, "invalid_status", "force constraint failure")
	})
	if err == nil {
		t.Fatal("expected history constraint failure")
	}
	_, err = repo.Get(ctx, id)
	if !errors.Is(err, domain.ErrTripNotFound) {
		t.Fatalf("trip remained after rollback: %v", err)
	}
}
