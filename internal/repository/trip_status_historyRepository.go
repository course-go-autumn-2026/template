package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"

	"github.com/poiuyttq/tripgo/internal/model"
	"github.com/poiuyttq/tripgo/internal/txmanager"
)

type TripStatusHistoryRepository struct{}

func NewTripStatusHistoryRepository() *TripStatusHistoryRepository {
	return &TripStatusHistoryRepository{}
}

func (r *TripStatusHistoryRepository) Add(ctx context.Context, history model.TripStatusHistory) error {
	queryBuilder := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	query, args, err := queryBuilder.
		Insert("trip_status_history").
		Columns(
			"trip_id",
			"from_status",
			"to_status",
			"reason",
			"changed_at",
		).
		Values(
			history.TripID,
			history.FromStatus,
			history.ToStatus,
			history.Reason,
			history.ChangedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build history insert query: %w", err)
	}

	if tx := txmanager.TxFromContext(ctx); tx != nil {
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("insert trip status history: %w", err)
		}
		return nil
	}

	return fmt.Errorf("trip status history repository requires transaction context")
}
