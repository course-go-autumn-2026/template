//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package listpositions_test
package listpositions

import (
	"context"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type tripRepository interface {
	GetTrip(context.Context, uuid.UUID) (trip.Trip, error)
	ListPositions(context.Context, uuid.UUID) ([]trip.Position, error)
}
