//go:build integration

package trip_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	"github.com/ggwpfu9/courseAvito/internal/repository/trip"
)

func setupDB(t *testing.T) *pgxpool.Pool {
	_ = godotenv.Load("../../../.env") // загружаем .env из корня проекта
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}
	return pool
}

func TestRepo_SaveAndGet(t *testing.T) {
	pool := setupDB(t)
	defer pool.Close()

	repo := trip.NewRepo(pool)
	ctx := context.Background()

	testTrip := appmodel.Trip{
		ID:       "11111111-1111-1111-1111-111111111111",
		UserID:   "22222222-2222-2222-2222-222222222222",
		DriverID: "33333333-3333-3333-3333-333333333333",
		Price:    1000,
		Status:   "active",
	}

	_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", testTrip.ID)
	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM trips WHERE id = $1", testTrip.ID)
	}()

	err := repo.Save(ctx, testTrip)
	if err != nil {
		t.Fatalf("failed to save trip: %v", err)
	}

	savedTrip, err := repo.GetByID(ctx, testTrip.ID)
	if err != nil {
		t.Fatalf("failed to get trip: %v", err)
	}

	if savedTrip.ID != testTrip.ID {
		t.Errorf("expected ID %s, got %s", testTrip.ID, savedTrip.ID)
	}
}