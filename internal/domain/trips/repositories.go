package trips

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateTrip(ctx context.Context, trip Trip) error
	GetTripByTripId(ctx context.Context, tripID uuid.UUID) (*Trip, error)
	FinishTrip(ctx context.Context, dto FinishTripDTO) (*Trip, error)
}

type FinishTripDTO struct {
	TripID     uuid.UUID
	FinishedAt time.Time
}
