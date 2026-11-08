package finish

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish
type usecase interface {
	Execute(context.Context, uuid.UUID) (domain.Trip, error)
}
