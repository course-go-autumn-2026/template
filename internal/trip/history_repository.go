package trip

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/database"
)

type HistoryRepository struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewHistoryRepository(pool *pgxpool.Pool, timeout time.Duration) *HistoryRepository {
	return &HistoryRepository{pool: pool, timeout: timeout}
}

func (r *HistoryRepository) Append(ctx context.Context, tripID uuid.UUID, from *api.TripStatus, to api.TripStatus, changedAt time.Time) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	query, args, err := sq.Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "changed_at").
		Values(tripID, from, to, changedAt).PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("build append trip history query: %w", err)
	}
	if _, err := database.ExecutorFromContext(ctx, r.pool).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("append trip history: %w", err)
	}
	return nil
}
