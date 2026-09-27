package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/models"
	"github.com/imrtlDrifter/template/internal/repositories/postgres"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Trip interface {
	Create(ctx context.Context, trip *models.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*models.Trip, error)
}

type StatusHistory interface {
	CreateStatusHistory(ctx context.Context, history *models.TripStatusHistory) error
}

type Idempotency interface {
	DeleteExpired(ctx context.Context) error
	TryCreate(ctx context.Context, key uuid.UUID, requestHash []byte, expiresAt time.Time) (bool, error)
	Get(ctx context.Context, key uuid.UUID) (*models.IdempotencyRecord, error)
	BindTrip(ctx context.Context, key, tripID uuid.UUID) error
}

type Repository struct {
	Trip          Trip
	StatusHistory StatusHistory
	Idempotency   Idempotency
}

func NewRepository(db *pgxpool.Pool, queryTimeout time.Duration) *Repository {
	return &Repository{
		Trip:          postgres.NewTripRepository(db, queryTimeout),
		StatusHistory: postgres.NewStatusHistoryRepository(db, queryTimeout),
		Idempotency:   postgres.NewIdempotencyRepository(db, queryTimeout),
	}
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func IsDriverBusyError(err error) bool {
	var pgErr *pgconn.PgError

	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "trips_one_active_per_driver_idx"
}
