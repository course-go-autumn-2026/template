//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package finish_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type tripRepository interface {
	GetTripForUpdate(context.Context, uuid.UUID) (trip.Trip, error)
	FinishTrip(context.Context, uuid.UUID, time.Time) error
	AddStatusHistory(context.Context, uuid.UUID, *trip.Status, trip.Status, string) error
}

type txManager interface {
	Do(context.Context, func(context.Context) error) error
}

type (
	clock interface{ Now() time.Time }

	observer interface{ TripCompleted(context.Context) }
)
