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

func setupDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	require.NotEmpty(t, dbURL, "DATABASE_URL is required")

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	cleanup := func() {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE trip_status_history, trips CASCADE")
		pool.Close()
	}
	return pool, cleanup
}

func TestRepository_RollbackOnSecondInsertError(t *testing.T) {
	pool, cleanup := setupDB(t)
	defer cleanup()

	ctx := context.Background()
	repo := repotrip.New(pool)
	txMgr := transaction.New(pool)

	tr := &model.Trip{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		StartPoint: model.Point{Latitude: 59.93, Longitude: 30.31},
		EndPoint:   model.Point{Latitude: 59.92, Longitude: 30.36},
		Price:      1450,
		Status:     model.StatusActive,
		StartedAt:  time.Now().UTC(),
	}

	err := txMgr.Do(ctx, func(txCtx context.Context) error {
		if err := repo.Save(txCtx, tr); err != nil {
			return err
		}
		return errors.New("simulated failure before history insert")
	})
	require.Error(t, err)

	_, err = repo.GetByID(ctx, tr.ID)
	require.ErrorIs(t, err, domaintrip.ErrTripNotFound)
}
