package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/Yolanesso/avito-labs/internal/model"
	"github.com/Yolanesso/avito-labs/internal/txmanager"
)

func (r *TripRepository) Finish(
	ctx context.Context,
	tripID uuid.UUID,
	finishedAt time.Time,
) error {
	ctx, cancel := context.WithTimeout(ctx, r.queryTimeout)
	defer cancel()

	builder := squirrel.Update("trips")

	builder = builder.Set(
		"status",
		string(model.TripStatusCompleted),
	)

	builder = builder.Set(
		"finished_at",
		finishedAt,
	)

	builder = builder.Set(
		"updated_at",
		finishedAt,
	)

	builder = builder.Where(squirrel.Eq{
		"id":     tripID,
		"status": string(model.TripStatusActive),
	})

	builder = builder.PlaceholderFormat(squirrel.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build finish trip query: %w", err)
	}

	executor := txmanager.ExecutorFromContext(ctx, r.pool)

	result, err := executor.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("finish trip: %w", err)
	}

	if result.RowsAffected() == 1 {
		return nil
	}

	_, err = r.GetByID(ctx, tripID)

	if errors.Is(err, model.ErrTripNotFound) {
		return model.ErrTripNotFound
	}

	if err != nil {
		return err
	}

	return model.ErrTripCompleted
}
