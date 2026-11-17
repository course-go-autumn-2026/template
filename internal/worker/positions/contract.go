//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package positions
package positions

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"

	"github.com/freyliee/template/internal/domain/trip"
)

type repository interface {
	FindStaleTrips(context.Context, time.Duration) ([]trip.StaleTrip, error)
	NeedsPosition(context.Context, uuid.UUID, time.Duration) (bool, error)
}

type gateway interface {
	RequestPosition(context.Context, uuid.UUID, uuid.UUID) error
}

type clock interface {
	Now() time.Time
	NewTicker(time.Duration) clockwork.Ticker
}

type observer interface {
	StaleCheck(context.Context, string)
	PositionRequest(context.Context, string, time.Duration)
}
