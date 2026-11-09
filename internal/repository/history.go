package repository

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/razrabbesezdav-art/template/internal/transaction"
)

// History сохраняет историю изменения статусов поездок.
type History struct {
	db           Querier
	builder      sq.StatementBuilderType
	queryTimeout time.Duration
}

func NewHistory(db Querier, queryTimeout time.Duration) *History {
	return &History{
		db:           db,
		builder:      sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		queryTimeout: queryTimeout,
	}
}

func (r *History) AddCreated(ctx context.Context, tripID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, nil, "active", "trip_created").
		ToSql()
	if err != nil {
		return fmt.Errorf("build trip creation history query: %w", err)
	}

	if _, err := r.queryer(ctx).Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert trip creation history: %w", err)
	}

	return nil
}

func (r *History) AddCompleted(ctx context.Context, tripID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	query, args, err := r.builder.
		Insert("trip_status_history").
		Columns("trip_id", "from_status", "to_status", "reason").
		Values(tripID, "active", "completed", "trip_completed").
		ToSql()
	if err != nil {
		return fmt.Errorf("build trip completion history query: %w", err)
	}

	if _, err := r.queryer(ctx).Exec(ctx, query, args...); err != nil {
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