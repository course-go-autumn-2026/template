//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/model"
)

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
}
