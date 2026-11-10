//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=contract.go -destination=mocks_test.go -package=finish_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type tripRepository interface {
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domaintrip.Trip, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type historyRepository interface {
	Append(ctx context.Context, tripID uuid.UUID, from *domaintrip.Status, to domaintrip.Status, changedAt time.Time) error
}
