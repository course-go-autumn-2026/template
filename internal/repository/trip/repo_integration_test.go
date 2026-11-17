//go:build integration

package trip_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/infra/postgres/transaction"
	repository "github.com/freyliee/template/internal/repository/trip"
)

func database(t *testing.T) (context.Context, *pgxpool.Pool, *repository.TripRepository, *transaction.TxManager) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(admin.Close)

	schema := "lab2_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{schema}.Sanitize()

	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()

		if _, err := admin.Exec(cleanup, "DROP SCHEMA "+identifier+" CASCADE"); err != nil {
			t.Errorf("cleanup schema: %v", err)
		}
	})

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}

	cfg.ConnConfig.RuntimeParams["search_path"] = schema

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(pool.Close)

	files, err := filepath.Glob("../../../migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no migrations found")
	}

	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		up, _, _ := strings.Cut(string(body), "-- +goose Down")

		if _, err := pool.Exec(ctx, up); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	return ctx, pool, repository.NewTripRepository(pool), transaction.NewTxManager(pool, 3*time.Second, logger)
}

func newTrip(t *testing.T) trip.Trip {
	t.Helper()

	entity, err := trip.New(uuid.New(), trip.Data{UserID: uuid.New(), DriverID: uuid.New(), Price: 321, StartPoint: trip.Coordinates{Latitude: 1, Longitude: 2}, EndPoint: trip.Coordinates{Latitude: 3, Longitude: 4}}, time.Now().UTC().Truncate(time.Microsecond))
	if err != nil {
		t.Fatal(err)
	}

	return entity
}

func TestRepositoryLifecycle(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	entity := newTrip(t)

	if err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.CreateTrip(ctx, entity); err != nil {
			return err
		}

		return repo.AddStatusHistory(ctx, entity.ID, nil, trip.StatusActive, "trip created")
	}); err != nil {
		t.Fatal(err)
	}

	actual, err := repo.GetTrip(ctx, entity.ID)
	if err != nil {
		t.Fatal(err)
	}

	if actual.ID != entity.ID || actual.Data != entity.Data || actual.Status != trip.StatusActive || !actual.StartedAt.Equal(entity.StartedAt) || actual.FinishedAt != nil {
		t.Fatalf("unexpected fetched trip: %+v", actual)
	}

	conflicting := newTrip(t)
	conflicting.DriverID = entity.DriverID

	if err := repo.CreateTrip(ctx, conflicting); !errors.Is(err, trip.ErrDriverBusy) {
		t.Fatalf("expected busy driver: %v", err)
	}

	at := entity.StartedAt.Add(time.Minute)

	if err := tx.Do(ctx, func(ctx context.Context) error {
		locked, err := repo.GetTripForUpdate(ctx, entity.ID)
		if err != nil {
			return err
		}

		previous := locked.Status

		if err := locked.Finish(at); err != nil {
			return err
		}

		if err := repo.FinishTrip(ctx, entity.ID, *locked.FinishedAt); err != nil {
			return err
		}

		return repo.AddStatusHistory(ctx, entity.ID, &previous, locked.Status, "trip finished")
	}); err != nil {
		t.Fatal(err)
	}

	actual, err = repo.GetTrip(ctx, entity.ID)
	if err != nil {
		t.Fatal(err)
	}

	if actual.Status != trip.StatusCompleted || actual.FinishedAt == nil || !actual.FinishedAt.Equal(at) {
		t.Fatalf("unexpected finished trip: %+v", actual)
	}

	var history int

	if err := pool.QueryRow(ctx, "SELECT count(*) FROM trip_status_history WHERE trip_id=$1", entity.ID).Scan(&history); err != nil || history != 2 {
		t.Fatalf("history count %d: %v", history, err)
	}

	if _, err := repo.GetTrip(ctx, uuid.New()); !errors.Is(err, trip.ErrTripNotFound) {
		t.Fatalf("expected not found: %v", err)
	}
}

func TestRollbackBothTables(t *testing.T) {
	ctx, pool, repo, tx := database(t)
	entity := newTrip(t)

	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := repo.CreateTrip(ctx, entity); err != nil {
			return err
		}

		if err := repo.AddStatusHistory(ctx, entity.ID, nil, trip.StatusActive, "trip created"); err != nil {
			return err
		}

		return repo.AddStatusHistory(ctx, entity.ID, nil, trip.Status("invalid"), "force constraint failure")
	})
	if err == nil {
		t.Fatal("constraint failure was not returned")
	}

	for _, table := range []string{"trips", "trip_status_history"} {
		var count int

		query := "SELECT count(*) FROM " + pgx.Identifier{table}.Sanitize()

		if err := pool.QueryRow(ctx, query).Scan(&count); err != nil || count != 0 {
			t.Fatalf("rollback left %d rows in %s: %v", count, table, err)
		}
	}
}
