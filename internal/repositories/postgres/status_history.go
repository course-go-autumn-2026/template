package postgres

import (
	"context"
	"time"

	"github.com/imrtlDrifter/template/internal/models"
	dbpostgres "github.com/imrtlDrifter/template/pkg/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StatusHistoryRepository interface {
	CreateStatusHistory(ctx context.Context, history *models.TripStatusHistory) error
}

type statusHistoryRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewStatusHistoryRepository(pool *pgxpool.Pool, queryTimeout time.Duration) StatusHistoryRepository {
	return &statusHistoryRepository{
		pool:         pool,
		queryTimeout: queryTimeout,
	}
}

func (r *statusHistoryRepository) CreateStatusHistory(ctx context.Context, history *models.TripStatusHistory) error {
	queryCtx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	err := dbpostgres.GetExecutor(queryCtx, r.pool).QueryRow(queryCtx,
		`INSERT INTO trip_status_history (trip_id, from_status, to_status, reason)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, changed_at`,
		history.TripID, history.FromStatus, history.ToStatus, history.Reason).Scan(&history.ID, &history.ChangedAt)
	if err != nil {
		return err
	}

	return nil
}
