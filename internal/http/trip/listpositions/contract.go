//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package listpositions_test
package listpositions

import (
	"context"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type usecase interface {
	Execute(context.Context, uuid.UUID) ([]trip.Position, error)
}
