//go:build integration

package trip

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	"github.com/whxtelyy/template/internal/infra/postgres/transaction"
	"github.com/whxtelyy/template/internal/usecase/trip/create"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestIdempotencyReplayAndConflict(t *testing.T) {
	pool := testPool(t)
	repo := NewTripRepository(pool, 5*time.Second)
	uc := create.New(repo, transaction.NewManager(pool, pgx.ReadCommitted, 5*time.Second), nil)
	key := uuid.New()
	req := create.Request{UserID: uuid.New(), DriverID: uuid.New(), Price: 100, IdempotencyKey: &key, RequestHash: "same"}
	ctx := context.Background()
	first, err := uc.Execute(ctx, req)
	if err != nil || first.Replayed {
		t.Fatalf("first create: %+v, %v", first, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM trip_idempotency_keys WHERE idempotency_key=$1", key)
		_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id=$1", first.Trip.ID)
	})
	second, err := uc.Execute(ctx, req)
	if err != nil || !second.Replayed || second.Trip.ID != first.Trip.ID {
		t.Fatalf("replay: %+v, %v", second, err)
	}
	req.RequestHash = "different"
	_, err = uc.Execute(ctx, req)
	if !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("conflict = %v", err)
	}
	var historyCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", first.Trip.ID).Scan(&historyCount); err != nil || historyCount != 1 {
		t.Fatalf("history count = %d, %v", historyCount, err)
	}
}

func TestConcurrentDriverBusy(t *testing.T) {
	pool := testPool(t)
	repo := NewTripRepository(pool, 5*time.Second)
	uc := create.New(repo, transaction.NewManager(pool, pgx.ReadCommitted, 5*time.Second), nil)
	driver := uuid.New()
	ctx := context.Background()
	results := make([]create.Result, 2)
	errs := make([]error, 2)
	var group sync.WaitGroup
	for i := range results {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			results[i], errs[i] = uc.Execute(ctx, create.Request{UserID: uuid.New(), DriverID: driver})
		}(i)
	}
	group.Wait()
	var success, busy int
	for i, err := range errs {
		if err == nil {
			success++
			_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id=$1", results[i].Trip.ID)
		}
		if errors.Is(err, domain.ErrDriverBusy) {
			busy++
		}
	}
	if success != 1 || busy != 1 {
		t.Fatalf("success=%d, driver_busy=%d, errors=%v", success, busy, errs)
	}
}
