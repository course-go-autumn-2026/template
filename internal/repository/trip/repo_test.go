//go:build integration

package trip_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/model"
	repotrip "github.com/xotapy-main/courseGo/internal/repository/trip"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, dbURL, "DATABASE_URL is required")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})
	return pool
}

func cleanupTrips(t *testing.T, pool *pgxpool.Pool, tripIDs ...uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		ctx := context.Background()
		for _, id := range tripIDs {
			_, _ = pool.Exec(ctx, "DELETE FROM trip_status_history WHERE trip_id = $1", id)
			_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", id)
		}
	})
}

func TestRepository_RollbackBothTablesOnError(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	repo := repotrip.New(pool)
	txMgr := transaction.New(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	tr := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1450,
		Status:     model.StatusActive,
		StartedAt:  now,
	}
	cleanupTrips(t, pool, tr.ID)

	err := txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Save(txCtx, tr); err != nil {
			return err
		}
		if err := repo.InsertHistory(txCtx, &model.TripStatusHistory{
			TripID:    tr.ID,
			Status:    string(tr.Status),
			ChangedAt: now,
		}); err != nil {
			return err
		}
		return errors.New("simulated failure after both inserts")
	})
	require.Error(t, err)

	// Проверяем, что в таблице trips записи нет
	_, err = repo.GetByID(ctx, tr.ID)
	require.ErrorIs(t, err, domaintrip.ErrTripNotFound)

	// Проверяем, что в таблице trip_status_history записей тоже нет
	var historyCount int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM trip_status_history WHERE trip_id = $1", tr.ID).Scan(&historyCount)
	require.NoError(t, err)
	require.Equal(t, 0, historyCount)
}

func TestRepository_SaveGetAndFinish(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	repo := repotrip.New(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)
	tr := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   model.Point{Latitude: 55.76, Longitude: 37.62},
		Price:      900,
		Status:     model.StatusActive,
		StartedAt:  now,
	}
	cleanupTrips(t, pool, tr.ID)

	require.NoError(t, repo.Save(ctx, tr))
	require.NoError(t, repo.InsertHistory(ctx, &model.TripStatusHistory{
		TripID:    tr.ID,
		Status:    string(tr.Status),
		ChangedAt: now,
	}))

	fetched, err := repo.GetByID(ctx, tr.ID)
	require.NoError(t, err)
	require.Equal(t, tr.ID, fetched.ID)
	require.Equal(t, model.StatusActive, fetched.Status)

	finishAt := now.Add(10 * time.Minute)
	finished, err := repo.Finish(ctx, tr.ID, finishAt)
	require.NoError(t, err)
	require.Equal(t, model.StatusCompleted, finished.Status)
	require.NotNil(t, finished.FinishedAt)

	// Повторный Finish должен вернуть ErrTripCompleted
	_, err = repo.Finish(ctx, tr.ID, finishAt.Add(time.Minute))
	require.ErrorIs(t, err, domaintrip.ErrTripCompleted)
}

func TestRepository_DriverBusyOnDuplicateActiveTrip(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	repo := repotrip.New(pool)

	driverID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	trip1 := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now,
	}
	trip2 := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1200,
		Status:     model.StatusActive,
		StartedAt:  now,
	}
	cleanupTrips(t, pool, trip1.ID, trip2.ID)

	require.NoError(t, repo.Save(ctx, trip1))

	err := repo.Save(ctx, trip2)
	require.ErrorIs(t, err, domaintrip.ErrDriverBusy)
}
