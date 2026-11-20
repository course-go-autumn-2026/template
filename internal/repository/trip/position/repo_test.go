//go:build integration

package position_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/infra/postgres/transaction"
	triprepo "github.com/George1487/MalyginGeorge/internal/repository/trip"
	posrepo "github.com/George1487/MalyginGeorge/internal/repository/trip/position"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// #nosec G101 -- default local dev postgres connection string for test environment
		dbURL = "postgres://tripgo:tripgo@localhost:23032/tripgo?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	err = pool.Ping(ctx)
	require.NoError(t, err, "database must be reachable for integration tests")

	t.Cleanup(func() {
		pool.Close()
	})

	return pool
}

func TestPositionRepository_AddPosition_and_UpdateLastPositionAt(t *testing.T) {
	pool := getTestPool(t)
	tripRepository := triprepo.New(pool)
	positionRepository := posrepo.New(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond).UTC()
	testTrip := trip.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      1000,
		Status:     trip.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}

	err := tripRepository.Save(ctx, testTrip)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", testTrip.ID)
	})

	t1 := now.Add(-5 * time.Minute)
	t2 := now.Add(-3 * time.Minute)
	t3 := now.Add(-1 * time.Minute)

	p2 := &trip.Position{
		TripID:      testTrip.ID,
		Coordinates: trip.Coordinates{Latitude: 55.751, Longitude: 37.611},
		RecordedAt:  t2,
	}
	savedP2, err := positionRepository.AddPosition(ctx, p2)
	require.NoError(t, err)
	require.NotZero(t, savedP2.ID)

	err = positionRepository.UpdateLastPositionAt(ctx, testTrip.ID, t2)
	require.NoError(t, err)

	tripAfterP2, err := tripRepository.GetTrip(ctx, testTrip.ID)
	require.NoError(t, err)
	require.NotNil(t, tripAfterP2.LastPositionAt)
	require.Equal(t, t2.Unix(), tripAfterP2.LastPositionAt.Unix())

	p1 := &trip.Position{
		TripID:      testTrip.ID,
		Coordinates: trip.Coordinates{Latitude: 55.750, Longitude: 37.610},
		RecordedAt:  t1,
	}
	savedP1, err := positionRepository.AddPosition(ctx, p1)
	require.NoError(t, err)
	require.NotZero(t, savedP1.ID)

	err = positionRepository.UpdateLastPositionAt(ctx, testTrip.ID, t1)
	require.NoError(t, err)

	tripAfterP1, err := tripRepository.GetTrip(ctx, testTrip.ID)
	require.NoError(t, err)
	require.NotNil(t, tripAfterP1.LastPositionAt)
	require.Equal(t, t2.Unix(), tripAfterP1.LastPositionAt.Unix(), "older coordinate must not age trip backwards")

	p3 := &trip.Position{
		TripID:      testTrip.ID,
		Coordinates: trip.Coordinates{Latitude: 55.752, Longitude: 37.612},
		RecordedAt:  t3,
	}
	savedP3, err := positionRepository.AddPosition(ctx, p3)
	require.NoError(t, err)
	require.NotZero(t, savedP3.ID)

	err = positionRepository.UpdateLastPositionAt(ctx, testTrip.ID, t3)
	require.NoError(t, err)

	tripAfterP3, err := tripRepository.GetTrip(ctx, testTrip.ID)
	require.NoError(t, err)
	require.NotNil(t, tripAfterP3.LastPositionAt)
	require.Equal(t, t3.Unix(), tripAfterP3.LastPositionAt.Unix())
}

func TestPositionRepository_Transaction_Rollback(t *testing.T) {
	pool := getTestPool(t)
	tripRepository := triprepo.New(pool)
	positionRepository := posrepo.New(pool)
	txManager := transaction.NewPgxTxManager(pool)
	ctx := context.Background()

	now := time.Now().Truncate(time.Microsecond).UTC()
	testTrip := trip.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: trip.Coordinates{Latitude: 55.75, Longitude: 37.61},
		EndPoint:   trip.Coordinates{Latitude: 55.76, Longitude: 37.62},
		Price:      1000,
		Status:     trip.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}

	err := tripRepository.Save(ctx, testTrip)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM trips WHERE id = $1", testTrip.ID)
	})

	posTime := now.Add(-2 * time.Minute)
	p := &trip.Position{
		TripID:      testTrip.ID,
		Coordinates: trip.Coordinates{Latitude: 55.755, Longitude: 37.615},
		RecordedAt:  posTime,
	}

	expectedErr := errors.New("simulated error before commit")
	txErr := txManager.Do(ctx, func(txCtx context.Context) error {
		_, addErr := positionRepository.AddPosition(txCtx, p)
		if addErr != nil {
			return addErr
		}

		if updErr := positionRepository.UpdateLastPositionAt(txCtx, testTrip.ID, posTime); updErr != nil {
			return updErr
		}

		return expectedErr
	})

	require.Error(t, txErr)
	require.ErrorIs(t, txErr, expectedErr)

	positions, err := positionRepository.GetPositions(ctx, testTrip.ID)
	require.NoError(t, err)
	require.Empty(t, positions, "positions should be empty due to transaction rollback")

	tripAfterRollback, err := tripRepository.GetTrip(ctx, testTrip.ID)
	require.NoError(t, err)
	require.Nil(t, tripAfterRollback.LastPositionAt, "last_position_at should remain nil due to rollback")
}
