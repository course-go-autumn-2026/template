package trip

import (
	"context"
	"fmt"
)

func (t *TripRepository) CountActive(ctx context.Context) (int64, error) {
	var count int64

	if err := t.pool.QueryRow(ctx, "SELECT count(*) FROM trips WHERE status = $1", "active").Scan(&count); err != nil {
		return 0, fmt.Errorf("count active trips: %w", err)
	}

	return count, nil
}
