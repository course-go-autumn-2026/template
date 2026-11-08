package finish

import (
	"context"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=finish
type tripRepository interface {
	Get(context.Context, uuid.UUID) (domain.Trip, error)
	Finish(context.Context, uuid.UUID) (domain.Trip, error)
	AddStatusHistory(context.Context, uuid.UUID, *string, string, string) error
}
type txManager interface {
	Do(context.Context, func(context.Context) error) error
}

type completionRecorder interface{ RecordCompleted(context.Context) }
