package model

import (
	"time"

	"github.com/google/uuid"
)

type CreatePosition struct {
	ID         int64
	TripID     uuid.UUID
	Longitude  float64
	Latitude   float64
	RecordedAt time.Time
}
