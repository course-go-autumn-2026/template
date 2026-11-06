//go:build integration

package trip_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/toooriii/tori/internal/config"
	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/infra/postgres"
	"github.com/toooriii/tori/internal/infra/postgres/transaction"
	"github.com/toooriii/tori/internal/model"
	repo "github.com/toooriii/tori/internal/repository/trip"
)

func setup(t *testing.T) *transaction.Manager {
	t.Helper()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	ctx := context.Background()
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	_, err = pool.Exec(ctx, "TRUNCATE trips, trip_status_history, idempotency_keys RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	t.Cleanup(func() { pool.Close() })

	tx := transaction.New(pool, pgx.ReadCommitted)
	return tx
}

func TestTripRepository_Save_And_Get(t *testing.T) {
	tx := setup(t)
	repo := repo.New(tx)
	ctx := context.Background()

	tr := model.Trip{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Start:     model.Point{Latitude: 1, Longitude: 1},
		End:       model.Point{Latitude: 2, Longitude: 2},
		Price:     100,
		Status:    trip.StatusActive,
		StartedAt: time.Now().UTC(),
	}

	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := repo.Get(ctx, tr.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != tr.ID {
		t.Fatalf("id = %v, want %v", got.ID, tr.ID)
	}
}

func TestTripRepository_DriverBusy(t *testing.T) {
	tx := setup(t)
	repo := repo.New(tx)
	ctx := context.Background()

	driverID := uuid.New()
	tr := model.Trip{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		DriverID:  driverID,
		Start:     model.Point{Latitude: 1, Longitude: 1},
		End:       model.Point{Latitude: 2, Longitude: 2},
		Price:     100,
		Status:    trip.StatusActive,
		StartedAt: time.Now().UTC(),
	}

	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("first save: %v", err)
	}

	tr2 := tr
	tr2.ID = uuid.New()
	err := repo.Save(ctx, tr2)
	if !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("err = %v, want ErrDriverBusy", err)
	}
}

func TestTripRepository_Transaction_Rollback(t *testing.T) {
	tx := setup(t)
	repo := repo.New(tx)
	ctx := context.Background()

	tr := model.Trip{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Start:     model.Point{Latitude: 1, Longitude: 1},
		End:       model.Point{Latitude: 2, Longitude: 2},
		Price:     100,
		Status:    trip.StatusActive,
		StartedAt: time.Now().UTC(),
	}

	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.Save(ctx, tr); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	_, err = repo.Get(ctx, tr.ID)
	if !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("trip should not exist after rollback, got err = %v", err)
	}
}
