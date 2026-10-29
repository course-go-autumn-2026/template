package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/razrabbesezdav-art/template/internal/transaction"
)

// History сохраняет историю изменения статусов поездок.
type History struct {
	db Querier
}

func NewHistory(db Querier) *History {
	return &History{db: db}
}

func (r *History) AddCreated(ctx context.Context, tripID uuid.UUID) error {
	query := `
		INSERT INTO trip_status_history (trip_id, from_status, to_status, reason)
		VALUES ($1, NULL, $2, $3)
	`

	_, err := r.queryer(ctx).Exec(
		ctx, query, tripID, "active", "trip_created",
	)
	if err != nil {
		return fmt.Errorf("insert trip creation history: %w", err)
	}

	return nil
}

func (r *History) AddCompleted(ctx context.Context, tripID uuid.UUID) error {
	query := `
		INSERT INTO trip_status_history (trip_id, from_status, to_status, reason)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.queryer(ctx).Exec(
		ctx, query, tripID, "active", "completed", "trip_completed",
	)
	if err != nil {
		return fmt.Errorf("insert trip completion history: %w", err)
	}

	return nil
}

func (r *History) queryer(ctx context.Context) Querier {
	if tx, ok := transaction.TxFromContext(ctx); ok {
		return tx
	}

	return r.db
}
