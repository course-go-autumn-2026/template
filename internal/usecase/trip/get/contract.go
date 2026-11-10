//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=contract.go -destination=mocks_test.go -package=get_test
package get

import (
	"context"

	"github.com/google/uuid"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}
