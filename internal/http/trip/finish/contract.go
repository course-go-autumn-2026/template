//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/eternal-despair/template/internal/model"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error)
}
