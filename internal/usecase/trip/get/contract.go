package get

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=get
type tripRepository interface {
	Get(context.Context, uuid.UUID) (domain.Trip, error)
}
