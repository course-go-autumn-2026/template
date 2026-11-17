package trip

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Data struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

type Trip struct {
	ID uuid.UUID
	Data
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func New(id uuid.UUID, data Data, at time.Time) (Trip, error) {
	if err := data.Validate(); err != nil {
		return Trip{}, err
	}

	return Trip{ID: id, Data: data, Status: StatusActive, StartedAt: at.UTC()}, nil
}

func (d Data) Validate() error {
	if d.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Message: "must be a non-zero UUID"}
	}

	if d.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Message: "must be a non-zero UUID"}
	}

	if d.Price < 0 {
		return &ValidationError{Field: "price", Message: "must be non-negative"}
	}

	for _, p := range []struct {
		name  string
		point Coordinates
	}{{"start_point", d.StartPoint}, {"end_point", d.EndPoint}} {
		if !coordinate(p.point.Latitude, -90, 90) {
			return &ValidationError{Field: p.name + ".latitude", Message: "must be between -90 and 90"}
		}

		if !coordinate(p.point.Longitude, -180, 180) {
			return &ValidationError{Field: p.name + ".longitude", Message: "must be between -180 and 180"}
		}
	}

	return nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}

	at = at.UTC()
	t.Status = StatusCompleted
	t.FinishedAt = &at

	return nil
}

func coordinate(v, min, max float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= min && v <= max
}
