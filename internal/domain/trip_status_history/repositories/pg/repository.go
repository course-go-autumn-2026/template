package pg

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	history "github.com/makkssik/template/internal/domain/trip_status_history"
)

type statusHistoryRepository struct {
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewPostgresqlRepository(pool *pgxpool.Pool, queryTimeout time.Duration) history.Repository {
	return &statusHistoryRepository{pool: pool, queryTimeout: queryTimeout}
}
