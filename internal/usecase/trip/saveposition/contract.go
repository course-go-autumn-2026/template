//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package saveposition_test
package saveposition

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type tripRepository interface {
	GetTripForUpdate(context.Context, uuid.UUID) (trip.Trip, error)
	SavePosition(context.Context, uuid.UUID, trip.PositionData) (trip.Position, error)
}

type txManager interface {
	Do(context.Context, func(context.Context) error) error
}

type (
	clock     interface{ Now() time.Time }
	canceller interface{ Cancel(uuid.UUID) }
	observer  interface{ PositionSaved(context.Context) }
)
