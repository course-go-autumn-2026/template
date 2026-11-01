package trip_status_history

import "context"

type Repository interface {
	RegisterStatusChange(ctx context.Context, change StatusChange) error
}
