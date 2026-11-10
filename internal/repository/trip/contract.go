package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type TripRepository interface {
	Create(ctx context.Context, value domaintrip.Trip) (domaintrip.Trip, error)
	Get(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (domaintrip.Trip, error)
}
