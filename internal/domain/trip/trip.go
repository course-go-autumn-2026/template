package trip

import (
	"time"

	"github.com/google/uuid"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Coordinates
	EndPoint       Coordinates
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func ValidateCoordinates(c Coordinates, prefix string) error {
	if c.Latitude < -90 || c.Latitude > 90 {
		return NewValidationError(prefix+".latitude", "latitude must be between -90 and 90")
	}
	if c.Longitude < -180 || c.Longitude > 180 {
		return NewValidationError(prefix+".longitude", "longitude must be between -180 and 180")
	}
	return nil
}

func NewTrip(userID, driverID uuid.UUID, startPoint, endPoint Coordinates, price int64, now time.Time) (*Trip, error) {
	if userID == uuid.Nil {
		return nil, NewValidationError("user_id", "user_id cannot be empty")
	}
	if driverID == uuid.Nil {
		return nil, NewValidationError("driver_id", "driver_id cannot be empty")
	}
	if err := ValidateCoordinates(startPoint, "start_point"); err != nil {
		return nil, err
	}
	if err := ValidateCoordinates(endPoint, "end_point"); err != nil {
		return nil, err
	}
	if price < 0 {
		return nil, NewValidationError("price", "price cannot be negative")
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
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	t.UpdatedAt = at
	return nil
}
