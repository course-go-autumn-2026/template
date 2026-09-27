package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/models"
	dbpostgres "github.com/imrtlDrifter/template/pkg/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TripRepository interface {
	Create(ctx context.Context, trip *models.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*models.Trip, error)
}

type tripRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewTripRepository(pool *pgxpool.Pool, queryTimeout time.Duration) TripRepository {
	return &tripRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *tripRepository) Create(ctx context.Context, trip *models.Trip) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	_, err := dbpostgres.GetExecutor(queryCtx, r.pool).Exec(queryCtx,
		`INSERT INTO trips (
			id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		trip.ID, trip.UserID, trip.DriverID, trip.StartPoint.Latitude,
		trip.StartPoint.Longitude, trip.EndPoint.Latitude, trip.EndPoint.Longitude,
		trip.Price, trip.Status, trip.StartedAt, trip.FinishedAt,
	)
	return err
}

func (r *tripRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var trip models.Trip
	err := dbpostgres.GetExecutor(queryCtx, r.pool).QueryRow(queryCtx,
		`SELECT id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at
		 FROM trips
		 WHERE id = $1`,
		id,
	).Scan(&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude, &trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude, &trip.Price, &trip.Status,
		&trip.StartedAt, &trip.FinishedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}

func (r *tripRepository) Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*models.Trip, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	var trip models.Trip
	err := dbpostgres.GetExecutor(queryCtx, r.pool).QueryRow(queryCtx,
		`UPDATE trips
		 SET status = $1, finished_at = $2, updated_at = $2
		 WHERE id = $3 AND status = $4
		 RETURNING id, user_id, driver_id,
			start_latitude, start_longitude,
			end_latitude, end_longitude,
			price, status, started_at, finished_at`,
		models.TripStatusCompleted, finishedAt, id, models.TripStatusActive,
	).Scan(&trip.ID, &trip.UserID, &trip.DriverID,
		&trip.StartPoint.Latitude, &trip.StartPoint.Longitude, &trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude, &trip.Price, &trip.Status,
		&trip.StartedAt, &trip.FinishedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}
