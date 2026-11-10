//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -source=contract.go -destination=mocks_test.go -package=create_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type tripRepository interface {
	Create(ctx context.Context, value domaintrip.Trip) (domaintrip.Trip, error)
}

type txManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type historyRepository interface {
	Append(ctx context.Context, tripID uuid.UUID, from *domaintrip.Status, to domaintrip.Status, changedAt time.Time) error
}

type idempotencyRepository interface {
	Claim(ctx context.Context, key uuid.UUID, hash []byte, expiresAt time.Time) (bool, error)
	Replay(ctx context.Context, key uuid.UUID, hash []byte) (domaintrip.Trip, error)
	Save(ctx context.Context, key uuid.UUID, value domaintrip.Trip) error
	Prune(ctx context.Context, currentKey uuid.UUID) error
}
