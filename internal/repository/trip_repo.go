package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ggwpfu9/courseAvito/internal/domain"
)

type TripRepo struct {
	pool *pgxpool.Pool
}

func NewTripRepo(pool *pgxpool.Pool) *TripRepo {
	return &TripRepo{pool: pool}
}

func (r *TripRepo) Create(ctx context.Context, trip *domain.Trip) error {
	exec := GetExecutor(ctx, r.pool)

	query, args, err := sq.Insert("trips").
		Columns("id", "user_id", "driver_id", "start_lat", "start_lon", "end_lat", "end_lon", "price", "status", "started_at").
		Values(trip.ID, trip.UserID, trip.DriverID, trip.StartPoint.Latitude, trip.StartPoint.Longitude, trip.EndPoint.Latitude, trip.EndPoint.Longitude, trip.Price, trip.Status, trip.StartedAt).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrDriverBusy
		}
		return fmt.Errorf("insert trip: %w", err)
	}

	historyQuery, historyArgs, err := sq.Insert("trip_status_history").
		Columns("trip_id", "status", "changed_at").
		Values(trip.ID, trip.Status, time.Now()).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	_, err = exec.Exec(ctx, historyQuery, historyArgs...)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	return nil
}

func (r *TripRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Trip, error) {
	exec := GetExecutor(ctx, r.pool)

	query, args, _ := sq.Select("id", "user_id", "driver_id", "start_lat", "start_lon", "end_lat", "end_lon", "price", "status", "started_at", "finished_at", "last_position_at").
		From("trips").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	var t domain.Trip
	err := exec.QueryRow(ctx, query, args...).Scan(
		&t.ID, &t.UserID, &t.DriverID, &t.StartPoint.Latitude, &t.StartPoint.Longitude, &t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &t.Status, &t.StartedAt, &t.FinishedAt, &t.LastPositionAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTripNotFound
		}
		return nil, fmt.Errorf("get trip: %w", err)
	}

	return &t, nil
}

func (r *TripRepo) Finish(ctx context.Context, id uuid.UUID) error {
	exec := GetExecutor(ctx, r.pool)

	query, args, err := sq.Update("trips").
		Set("status", "completed").
		Set("finished_at", time.Now()).
		Where(sq.Eq{"id": id, "status": "active"}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	res, err := exec.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}

	if res.RowsAffected() == 0 {
		trip, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}
		if trip.Status == "completed" {
			return domain.ErrTripCompleted
		}
		return domain.ErrTripNotFound
	}

	return nil
}