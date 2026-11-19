//go:build integration

package trip_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/model"
	repotrip "github.com/xotapy-main/courseGo/internal/repository/trip"
)

func TestRepository_FindStaleActiveTrips(t *testing.T) {
	pool := setupDB(t)
	ctx := context.Background()
	repo := repotrip.New(pool)

	now := time.Now().UTC().Truncate(time.Microsecond)

	staleNoPos := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now.Add(-5 * time.Minute),
	}
	staleWithOldPos := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}
	freshTrip := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}
	completedTrip := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}
	cleanupTrips(t, pool, staleNoPos.ID, staleWithOldPos.ID, freshTrip.ID, completedTrip.ID)

	require.NoError(t, repo.Save(ctx, staleNoPos))
	require.NoError(t, repo.Save(ctx, staleWithOldPos))
	require.NoError(t, repo.Save(ctx, freshTrip))
	require.NoError(t, repo.Save(ctx, completedTrip))

	require.NoError(t, repo.UpdateLastPositionAt(ctx, staleWithOldPos.ID, now.Add(-3*time.Minute)))
	require.NoError(t, repo.UpdateLastPositionAt(ctx, freshTrip.ID, now.Add(-10*time.Second)))
	_, err := repo.Finish(ctx, completedTrip.ID, now.Add(-2*time.Minute))
	require.NoError(t, err)

	stale, err := repo.FindStaleActiveTrips(ctx, time.Minute)
	require.NoError(t, err)

	foundIDs := make([]uuid.UUID, 0, len(stale))
	for _, item := range stale {
		if item.ID == staleNoPos.ID || item.ID == staleWithOldPos.ID || item.ID == freshTrip.ID || item.ID == completedTrip.ID {
			foundIDs = append(foundIDs, item.ID)
		}
	}

	require.Equal(t, []uuid.UUID{staleNoPos.ID, staleWithOldPos.ID}, foundIDs)
}

func TestRepository_SavePositionOutOfOrderAndRollback(t *testing.T) {
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
		Price:      1000,
		Status:     model.StatusActive,
		StartedAt:  now.Add(-10 * time.Minute),
	}
	cleanupTrips(t, pool, tr.ID)
	require.NoError(t, repo.Save(ctx, tr))

	tNewer := now.Add(-1 * time.Minute)
	tOlder := now.Add(-4 * time.Minute)

	err := txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.UpdateLastPositionAt(txCtx, tr.ID, tNewer); err != nil {
			return err
		}
		_, err := repo.InsertPosition(txCtx, &model.Position{
			TripID:      tr.ID,
			Coordinates: model.Point{Latitude: 59.94, Longitude: 30.34},
			RecordedAt:  tNewer,
		})
		if err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)

	fetchedAfterRollback, err := repo.GetByID(ctx, tr.ID)
	require.NoError(t, err)
	require.Nil(t, fetchedAfterRollback.LastPositionAt)

	positionsAfterRollback, err := repo.ListPositions(ctx, tr.ID)
	require.NoError(t, err)
	require.Empty(t, positionsAfterRollback)

	require.NoError(t, txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.UpdateLastPositionAt(txCtx, tr.ID, tNewer); err != nil {
			return err
		}
		_, err := repo.InsertPosition(txCtx, &model.Position{
			TripID:      tr.ID,
			Coordinates: model.Point{Latitude: 59.94, Longitude: 30.34},
			RecordedAt:  tNewer,
		})
		return err
	}))

	require.NoError(t, txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.UpdateLastPositionAt(txCtx, tr.ID, tOlder); err != nil {
			return err
		}
		_, err := repo.InsertPosition(txCtx, &model.Position{
			TripID:      tr.ID,
			Coordinates: model.Point{Latitude: 59.935, Longitude: 30.335},
			RecordedAt:  tOlder,
		})
		return err
	}))

	fetched, err := repo.GetByID(ctx, tr.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched.LastPositionAt)
	require.True(t, fetched.LastPositionAt.Equal(tNewer))

	positions, err := repo.ListPositions(ctx, tr.ID)
	require.NoError(t, err)
	require.Len(t, positions, 2)
	require.True(t, positions[0].RecordedAt.Equal(tOlder))
	require.True(t, positions[1].RecordedAt.Equal(tNewer))
}
