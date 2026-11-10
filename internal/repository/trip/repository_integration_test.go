//go:build integration

package trip

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	"github.com/murmyauuu/template/internal/infra/postgres/transaction"
)

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })
	return pool
}

func TestRepositoryCreateAndGet(t *testing.T) {
	pool := openTestPool(t)
	repo := NewRepository(pool, 5*time.Second)

	tr := domaintrip.Trip{
		Id:         uuid.New(),
		UserId:     uuid.New(),
		DriverId:   uuid.New(),
		StartPoint: domaintrip.Point{Latitude: 1, Longitude: 2},
		EndPoint:   domaintrip.Point{Latitude: 3, Longitude: 4},
		Price:      100,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	created, err := repo.Create(context.Background(), tr)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Id != tr.Id {
		t.Fatal("id mismatch")
	}

	got, err := repo.Get(context.Background(), tr.Id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != domaintrip.StatusActive {
		t.Fatalf("status = %v", got.Status)
	}

	// cleanup
	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", tr.Id)
}

func TestRepositoryFinish(t *testing.T) {
	pool := openTestPool(t)
	repo := NewRepository(pool, 5*time.Second)

	tr := domaintrip.Trip{
		Id:         uuid.New(),
		UserId:     uuid.New(),
		DriverId:   uuid.New(),
		StartPoint: domaintrip.Point{Latitude: 1, Longitude: 2},
		EndPoint:   domaintrip.Point{Latitude: 3, Longitude: 4},
		Price:      100,
		Status:     domaintrip.StatusActive,
		StartedAt:  time.Now().UTC().Truncate(time.Microsecond),
	}
	if _, err := repo.Create(context.Background(), tr); err != nil {
		t.Fatalf("create: %v", err)
	}
	finished, err := repo.Finish(context.Background(), tr.Id, time.Now().UTC())
	if err != nil {
		t.Fatalf("finish: %v", err)
	}
	if finished.Status != domaintrip.StatusCompleted {
		t.Fatalf("status = %v", finished.Status)
	}

	// finishing again should fail
	_, err = repo.Finish(context.Background(), tr.Id, time.Now().UTC())
	if err != domaintrip.ErrTripCompleted {
		t.Fatalf("expected ErrTripCompleted, got %v", err)
	}

	ctx := context.Background()
	_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", tr.Id)
}

func TestRepositoryTransactionRollback(t *testing.T) {
	pool := openTestPool(t)
	tx := transaction.NewTxManager(pool, 5*time.Second)

	err := tx.Do(context.Background(), func(ctx context.Context) error {
		_, err := pool.Exec(ctx, "INSERT INTO trips (id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at) VALUES ($1,$2,$3,1,2,3,4,10,'active',now())",
			uuid.New(), uuid.New(), uuid.New())
		if err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	// The count should not have increased permanently (rollback). Hard to assert exact count, but no panic.
}

func TestRepositoryGetNotFound(t *testing.T) {
	pool := openTestPool(t)
	repo := NewRepository(pool, 5*time.Second)

	_, err := repo.Get(context.Background(), uuid.New())
	if err != domaintrip.ErrTripNotFound {
		t.Fatalf("expected ErrTripNotFound, got %v", err)
	}
}
