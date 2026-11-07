package repository

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/XrystikLK/template/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTripNotFound         = errors.New("trip not found")
	ErrDriverBusy           = errors.New("driver busy")
	ErrTripAlreadyCompleted = errors.New("trip already completed")
)

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type StatusHistory struct {
	TripID     uuid.UUID
	FromStatus *string
	ToStatus   string
	Reason     *string
	ChangedAt  time.Time
}

type TripRepository struct {
	pool         *pgxpool.Pool
	sq           squirrel.StatementBuilderType
	queryTimeout time.Duration
}

func NewTripRepository(pool *pgxpool.Pool, queryTimeout time.Duration) *TripRepository {
	if queryTimeout <= 0 {
		queryTimeout = 3 * time.Second
	}
	return &TripRepository{
		pool:         pool,
		sq:           squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		queryTimeout: queryTimeout,
	}
}

func (r *TripRepository) CreateTrip(ctx context.Context, trip *Trip) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.sq.Insert("trips").
		Columns(
			"id", "user_id", "driver_id",
			"start_latitude", "start_longitude",
			"end_latitude", "end_longitude",
			"price", "status", "started_at",
			"created_at", "updated_at",
		).
		Values(
			trip.ID, trip.UserID, trip.DriverID,
			trip.StartLatitude, trip.StartLongitude,
			trip.EndLatitude, trip.EndLongitude,
			trip.Price, trip.Status, trip.StartedAt,
			trip.CreatedAt, trip.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return err
	}

	exec := db.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDriverBusy
		}
		return err
	}

	return nil
}

func (r *TripRepository) AddStatusHistory(ctx context.Context, history *StatusHistory) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason", "changed_at").
		Values(history.TripID, history.FromStatus, history.ToStatus, history.Reason, history.ChangedAt).
		ToSql()
	if err != nil {
		return err
	}

	exec := db.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	return err
}

func (r *TripRepository) GetTripByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.sq.Select(
		"id", "user_id", "driver_id",
		"start_latitude", "start_longitude",
		"end_latitude", "end_longitude",
		"price", "status", "started_at", "finished_at",
		"created_at", "updated_at",
	).
		From("trips").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, err
	}

	exec := db.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	var t Trip
	err = row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTripNotFound
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (r *TripRepository) FinishTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*Trip, error) {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	now := time.Now()
	query, args, err := r.sq.Update("trips").
		Set("status", "completed").
		Set("finished_at", finishedAt).
		Set("updated_at", now).
		Where(squirrel.Eq{"id": id, "status": "active"}).
		Suffix("RETURNING id, user_id, driver_id, start_latitude, start_longitude, end_latitude, end_longitude, price, status, started_at, finished_at, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, err
	}

	exec := db.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx, query, args...)

	var t Trip
	err = row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartLatitude, &t.StartLongitude,
		&t.EndLatitude, &t.EndLongitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err == nil {
		return &t, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		checkQuery, checkArgs, checkErr := r.sq.Select("status").From("trips").Where(squirrel.Eq{"id": id}).ToSql()
		if checkErr != nil {
			return nil, checkErr
		}

		var currentStatus string
		checkErr = exec.QueryRow(ctx, checkQuery, checkArgs...).Scan(&currentStatus)
		if errors.Is(checkErr, pgx.ErrNoRows) {
			return nil, ErrTripNotFound
		}
		if currentStatus == "completed" {
			return nil, ErrTripAlreadyCompleted
		}
		if checkErr != nil {
			return nil, checkErr
		}
	}

	return nil, err
}
