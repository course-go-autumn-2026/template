package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/Godofdeath-dev/avitoLabs/internal/generated"
)

var (
	ErrNotFound        = errors.New("trip not found") // todo попозже заменю на сгенерированные в api.gen
	ErrAlreadyFinished = errors.New("trip already completed")
)

type TripRepository struct {
	pool *pgxpool.Pool
}

func NewTripRepository(pool *pgxpool.Pool) *TripRepository {
	return &TripRepository{pool: pool}
}

// InsertTrip выполняет вставку только в таблицу trips
func (r *TripRepository) InsertTrip(ctx context.Context, trip api.Trip) error {
	executor := getExecutor(ctx, r.pool)

	query := `
		INSERT INTO trips (
			id, user_id, driver_id, price, status,
			start_latitude, start_longitude, end_latitude, end_longitude,
			started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := executor.Exec(ctx, query,
		trip.Id,
		trip.UserId,
		trip.DriverId,
		trip.Price,
		trip.Status,
		trip.StartPoint.Latitude,
		trip.StartPoint.Longitude,
		trip.EndPoint.Latitude,
		trip.EndPoint.Longitude,
		trip.StartedAt,
	)
	return err
}

// AddStatusHistory выполняет запись в таблицу trip_status_history
func (r *TripRepository) AddStatusHistory(ctx context.Context, tripID api.TripId, fromStatus *string, toStatus api.TripStatus, changedAt time.Time) error {
	executor := getExecutor(ctx, r.pool)

	query := `
		INSERT INTO trip_status_history (trip_id, from_status, to_status, changed_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := executor.Exec(ctx, query, tripID, fromStatus, string(toStatus), changedAt)
	return err
}

// GetTripByID читает поездку (в транзакции или через пул в зависимости от контекста)
func (r *TripRepository) GetTripByID(ctx context.Context, id api.TripId) (*api.Trip, error) {
	executor := getExecutor(ctx, r.pool)

	query := `
		SELECT 
			id, user_id, driver_id, price, status,
			start_latitude, start_longitude, end_latitude, end_longitude,
			started_at, finished_at
		FROM trips
		WHERE id = $1
	`

	var trip api.Trip
	err := executor.QueryRow(ctx, query, id).Scan(
		&trip.Id,
		&trip.UserId,
		&trip.DriverId,
		&trip.Price,
		&trip.Status,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.StartedAt,
		&trip.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &trip, nil
}

// UpdateTripCompleted переводит поездку в статус completed с блокировкой строки
func (r *TripRepository) UpdateTripCompleted(ctx context.Context, id api.TripId, finishedAt time.Time) (*api.Trip, error) {
	executor := getExecutor(ctx, r.pool)

	var currentStatus string
	queryCheck := `SELECT status FROM trips WHERE id = $1 FOR UPDATE`
	err := executor.QueryRow(ctx, queryCheck, id).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if currentStatus == string(api.Completed) {
		return nil, ErrAlreadyFinished
	}

	queryUpdate := `
		UPDATE trips
		SET status = $2, finished_at = $3, updated_at = $3
		WHERE id = $1
		RETURNING 
			id, user_id, driver_id, price, status,
			start_latitude, start_longitude, end_latitude, end_longitude,
			started_at, finished_at
	`

	var trip api.Trip
	err = executor.QueryRow(ctx, queryUpdate, id, string(api.Completed), finishedAt).Scan(
		&trip.Id,
		&trip.UserId,
		&trip.DriverId,
		&trip.Price,
		&trip.Status,
		&trip.StartPoint.Latitude,
		&trip.StartPoint.Longitude,
		&trip.EndPoint.Latitude,
		&trip.EndPoint.Longitude,
		&trip.StartedAt,
		&trip.FinishedAt,
	)
	if err != nil {
		return nil, err
	}

	return &trip, nil
}
