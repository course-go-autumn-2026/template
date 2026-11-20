package position

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

func (r *Repository) GetPositions(ctx context.Context, id uuid.UUID) ([]trip.Position, error) {
	query, args, err := sq.
		Select(
			"id",
			"trip_id",
			"latitude",
			"longitude",
			"recorded_at",
			"created_at",
		).
		From("trip_positions").
		Where(sq.Eq{"trip_id": id}).
		OrderBy("recorded_at ASC, id ASC").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select positions query: %w", err)
	}
	row, err := r.executor(ctx).Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("positions query row: %w", err)
	}
	defer row.Close()
	var positions []trip.Position
	for row.Next() {
		var position trip.Position
		position, err = scanPosition(row)
		if err != nil {
			return nil, fmt.Errorf("scan position: %w", err)
		}
		positions = append(positions, position)
	}
	if err := row.Err(); err != nil {
		return nil, fmt.Errorf("iterate positions rows: %w", err)
	}
	return positions, nil
}
