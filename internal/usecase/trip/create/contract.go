//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type tripRepository interface {
	CreateTrip(context.Context, trip.Trip) error
	AddStatusHistory(context.Context, uuid.UUID, *trip.Status, trip.Status, string) error
}

type txManager interface {
	Do(context.Context, func(context.Context) error) error
}

type (
	clock interface{ Now() time.Time }

	observer interface{ TripCreated(context.Context) }
)
