package create

import (
	"context"

	"github.com/whxtelyy/template/internal/usecase/trip/create"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create
type usecase interface {
	Execute(context.Context, create.Request) (create.Result, error)
}
