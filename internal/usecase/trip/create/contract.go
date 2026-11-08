package create

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=create
type tripRepository interface {
	Create(context.Context, domain.Trip) (domain.Trip, error)
	AddStatusHistory(context.Context, uuid.UUID, *string, string, string) error
	Get(context.Context, uuid.UUID) (domain.Trip, error)
	ReserveIdempotencyKey(context.Context, uuid.UUID, string) (domain.IdempotencyRecord, bool, error)
	SetIdempotencyTrip(context.Context, uuid.UUID, uuid.UUID) error
}

type txManager interface {
	Do(context.Context, func(context.Context) error) error
}

type creationRecorder interface{ RecordCreated(context.Context) }
