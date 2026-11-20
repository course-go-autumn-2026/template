// Package model holds the plain data structures passed between layers.
package model

import (
	"time"

	"github.com/google/uuid"
)

type Coordinates struct{ Latitude, Longitude float64 }

type Trip struct {
	ID, UserID, DriverID uuid.UUID
	StartPoint, EndPoint Coordinates
	Price                int64
	Status               string
	StartedAt            time.Time
	FinishedAt           *time.Time
	LastPositionAt       *time.Time
}
