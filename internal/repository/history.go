package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/Yolanesso/avito-labs/internal/txmanager"
)

func (r *TripRepository) AddCreatedHistory(
	ctx context.Context,
	tripID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	builder := squirrel.Insert("trip_status_history")

	builder = builder.Columns(
		"trip_id",
		"from_status",
		"to_status",
		"reason",
	)

	builder = builder.Values(
		tripID,
		nil,
		"active",
		"trip created",
	)

	builder = builder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build created history query: %w", err)
	}

	executor := txmanager.ExecutorFromContext(ctx, r.pool)

	_, err = executor.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add created history: %w", err)
	}

	return nil
}

func (r *TripRepository) AddCompletedHistory(
	ctx context.Context,
	tripID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	builder := squirrel.Insert("trip_status_history")

	builder = builder.Columns(
		"trip_id",
		"from_status",
		"to_status",
		"reason",
	)

	builder = builder.Values(
		tripID,
		"active",
		"completed",
		"trip finished",
	)

	builder = builder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build completed history query: %w", err)
	}

	executor := txmanager.ExecutorFromContext(ctx, r.pool)

	_, err = executor.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add completed history: %w", err)
	}

	return nil
}
