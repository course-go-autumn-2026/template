//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package saveposition_test
package saveposition

import (
	"context"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type usecase interface {
	Execute(context.Context, uuid.UUID, trip.PositionData) (trip.Position, error)
}
