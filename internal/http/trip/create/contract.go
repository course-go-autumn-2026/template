//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/eternal-despair/template/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, input model.CreateTripInput) (*model.Trip, error)
}
