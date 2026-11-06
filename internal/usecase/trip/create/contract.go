//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test

package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/model"
)

type (
	tripRepository interface {
		Save(ctx context.Context, t model.Trip) error
		AddHistory(ctx context.Context, tripID uuid.UUID, from *string, to, reason string) error
		Get(ctx context.Context, id uuid.UUID) (model.Trip, error)
	}

	idempotencyRepository interface {
		Save(ctx context.Context, key uuid.UUID, hash string, tripID uuid.UUID) error
		Get(ctx context.Context, key uuid.UUID) (model.IdempotencyRecord, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
