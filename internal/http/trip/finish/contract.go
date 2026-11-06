//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/model"
)

type UseCase interface {
	Do(ctx context.Context, id uuid.UUID, now time.Time) (model.Trip, error)
}
