//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/model"
)

type (
	tripRepository interface {
		Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
		Finish(ctx context.Context, id uuid.UUID, at time.Time) (model.Trip, error)
		AddHistory(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
