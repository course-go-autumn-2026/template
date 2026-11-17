package trip

import (
	"time"

	"github.com/google/uuid"
)

type PositionData struct {
	Coordinates Coordinates
	RecordedAt  time.Time
}

type Position struct {
	ID     int64
	TripID uuid.UUID
	PositionData
}

type StaleTrip struct {
	ID             uuid.UUID
	DriverID       uuid.UUID
	LastPositionAt time.Time
}

func (p PositionData) Validate(now time.Time) error {
	if !coordinate(p.Coordinates.Latitude, -90, 90) {
		return &ValidationError{Field: "coordinates.latitude", Message: "must be between -90 and 90"}
	}

	if !coordinate(p.Coordinates.Longitude, -180, 180) {
		return &ValidationError{Field: "coordinates.longitude", Message: "must be between -180 and 180"}
	}

	if p.RecordedAt.IsZero() || p.RecordedAt.After(now.Add(time.Minute)) {
		return &ValidationError{Field: "recorded_at", Message: "must be set and at most one minute in the future"}
	}

	return nil
}
