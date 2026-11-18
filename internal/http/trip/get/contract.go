//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/XrystikLK/template/internal/model"
)

type useCase interface {
	Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}
