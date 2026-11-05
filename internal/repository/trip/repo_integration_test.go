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
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/infra/postgres/transaction"
	"github.com/eternal-despair/template/internal/model"
	repoPkg "github.com/eternal-despair/template/internal/repository/trip"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *repoPkg.Repository, *transaction.Manager) {
	t.Helper()

	_ = godotenv.Load("../../../.env")
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set, skipping integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("cannot connect to test database at %s: %v", dbURL, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("cannot ping test database at %s: %v", dbURL, err)
	}

	repo := repoPkg.New(pool)
	txMgr := transaction.New(pool)

	return pool, repo, txMgr
}

func cleanupTrip(ctx context.Context, pool *pgxpool.Pool, tripID uuid.UUID) {
	_, _ = pool.Exec(ctx, "DELETE FROM trip_status_history WHERE trip_id = $1", tripID)
	_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", tripID)
}

func TestRepository_CreateAndGet(t *testing.T) {
	pool, repo, _ := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	tripID := uuid.New()
	driverID := uuid.New()
	userID := uuid.New()

	defer cleanupTrip(ctx, pool, tripID)

	tr := &model.Trip{
		ID:       tripID,
		UserID:   userID,
		DriverID: driverID,
		StartPoint: trip.Coordinates{
			Latitude:  55.751244,
			Longitude: 37.618423,
		},
		EndPoint: trip.Coordinates{
			Latitude:  55.755814,
			Longitude: 37.617635,
		},
		Price:     750,
		Status:    trip.StatusActive,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	err := repo.Create(ctx, tr)
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, tripID)
	require.NoError(t, err)
	assert.Equal(t, tripID, fetched.ID)
	assert.Equal(t, userID, fetched.UserID)
	assert.Equal(t, driverID, fetched.DriverID)
	assert.Equal(t, int64(750), fetched.Price)
	assert.Equal(t, trip.StatusActive, fetched.Status)
	assert.Nil(t, fetched.FinishedAt)
}

func TestRepository_DuplicateDriverActiveTrip(t *testing.T) {
	pool, repo, _ := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	driverID := uuid.New()

	trip1ID := uuid.New()
	trip2ID := uuid.New()

	defer cleanupTrip(ctx, pool, trip1ID)
	defer cleanupTrip(ctx, pool, trip2ID)

	tr1 := &model.Trip{
		ID:         trip1ID,
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      500,
		Status:     trip.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := repo.Create(ctx, tr1)
	require.NoError(t, err)

	tr2 := &model.Trip{
		ID:         trip2ID,
		UserID:     uuid.New(),
		DriverID:   driverID,
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      600,
		Status:     trip.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err = repo.Create(ctx, tr2)
	require.ErrorIs(t, err, trip.ErrDriverBusy)
}

func TestRepository_Finish(t *testing.T) {
	pool, repo, _ := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	tripID := uuid.New()

	defer cleanupTrip(ctx, pool, tripID)

	tr := &model.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      1000,
		Status:     trip.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := repo.Create(ctx, tr)
	require.NoError(t, err)

	finishTime := now.Add(15 * time.Minute)
	finished, err := repo.Finish(ctx, tripID, finishTime)
	require.NoError(t, err)
	assert.Equal(t, trip.StatusCompleted, finished.Status)
	require.NotNil(t, finished.FinishedAt)

	// Повторное завершение уже завершенной поездки возвращает ErrTripCompleted
	_, err = repo.Finish(ctx, tripID, finishTime.Add(5*time.Minute))
	assert.ErrorIs(t, err, trip.ErrTripCompleted)
}

func TestRepository_GetNotFound(t *testing.T) {
	pool, repo, _ := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	randomID := uuid.New()

	_, err := repo.GetByID(ctx, randomID)
	assert.ErrorIs(t, err, trip.ErrTripNotFound)
}

func TestRepository_TransactionRollback(t *testing.T) {
	pool, repo, txMgr := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	tripID := uuid.New()

	defer cleanupTrip(ctx, pool, tripID)

	tr := &model.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      800,
		Status:     trip.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	simulatedErr := errors.New("simulated error to trigger rollback")

	// 1. Проверяем откат по возврату ошибки
	err := txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Create(txCtx, tr); err != nil {
			return err
		}
		// Симулируем падение перед записью истории
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	// Поездка не должна была сохраниться в БД
	_, err = repo.GetByID(ctx, tripID)
	assert.ErrorIs(t, err, trip.ErrTripNotFound)

	// 2. Проверяем откат при панике
	assert.Panics(t, func() {
		_ = txMgr.Do(ctx, func(txCtx context.Context) error {
			_ = repo.Create(txCtx, tr)
			panic("unexpected runtime panic")
		})
	})

	// Поездка не должна была сохраниться в БД после паники
	_, err = repo.GetByID(ctx, tripID)
	assert.ErrorIs(t, err, trip.ErrTripNotFound)
}
