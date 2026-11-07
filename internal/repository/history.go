package repository

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/txmanager"
)

type HistoryRepository struct {
	pool *pgxpool.Pool
	sb   sq.StatementBuilderType
}

func NewHistoryRepository(pool *pgxpool.Pool) *HistoryRepository {
	return &HistoryRepository{
		pool: pool,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *HistoryRepository) Insert(ctx context.Context, record *model.TripStatusHistory) error {
	query, args, err := r.sb.Insert("trip_status_history").
		Columns("trip_id", "status", "changed_at").
		Values(record.TripID, record.Status, record.ChangedAt).
		ToSql()
	if err != nil {
		return err
	}

	exec := txmanager.GetExecutor(ctx, r.pool)
	_, err = exec.Exec(ctx, query, args...)
	return err
}
