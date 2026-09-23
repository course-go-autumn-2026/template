package trip

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive    = "active"
	StatusCompleted = "completed"
)

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartLat   float64
	StartLon   float64
	EndLat     float64
	EndLon     float64
	Price      int64
	Status     string
	StartedAt  time.Time
	FinishedAt *time.Time
}

type CreateInput struct {
	UserID   uuid.UUID
	DriverID uuid.UUID
	StartLat float64
	StartLon float64
	EndLat   float64
	EndLon   float64
	Price    int64
}

type IdempotencyRecord struct {
	Key         uuid.UUID
	RequestHash string
	TripID      uuid.UUID
	ExpiresAt   time.Time
}
