package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

func (p Point) Validate() error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return NewValidationError("latitude", "must be between -90 and 90")
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return NewValidationError("longitude", "must be between -180 and 180")
	}
	return nil
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Point
	EndPoint       Point
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func NewTrip(userID, driverID uuid.UUID, startPoint, endPoint Point, price int64, now time.Time) (*Trip, error) {
	if userID == uuid.Nil {
		return nil, NewValidationError("user_id", "must not be empty")
	}
	if driverID == uuid.Nil {
		return nil, NewValidationError("driver_id", "must not be empty")
	}
	if err := startPoint.Validate(); err != nil {
		return nil, err
	}
	if err := endPoint.Validate(); err != nil {
		return nil, err
	}
	if price < 0 {
		return nil, NewValidationError("price", "cannot be negative")
	}

	return &Trip{
		ID:         uuid.New(),
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: startPoint,
		EndPoint:   endPoint,
		Price:      price,
		Status:     StatusActive,
		StartedAt:  now,
	}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}
