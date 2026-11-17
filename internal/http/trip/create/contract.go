//go:generate go tool mockgen -source contract.go -destination mocks_test.go -package create_test
package create

import (
	"context"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/model"
)

type usecase interface {
	Execute(context.Context, model.CreateTripInput) (trip.Trip, error)
}
