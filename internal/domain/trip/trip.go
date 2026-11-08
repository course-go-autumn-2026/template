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

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func New(userID, driverID uuid.UUID, start, end Coordinates, price int64, now time.Time) (*Trip, error) {
	if userID == uuid.Nil {
		return nil, &ValidationError{Field: "user_id", Message: "cannot be empty"}
	}
	if driverID == uuid.Nil {
		return nil, &ValidationError{Field: "driver_id", Message: "cannot be empty"}
	}
	if price < 0 {
		return nil, &ValidationError{Field: "price", Message: "must be greater than or equal to 0"}
	}
	if start.Latitude < -90 || start.Latitude > 90 || start.Longitude < -180 || start.Longitude > 180 {
		return nil, &ValidationError{Field: "start_point", Message: "coordinates out of valid range"}
	}
	if end.Latitude < -90 || end.Latitude > 90 || end.Longitude < -180 || end.Longitude > 180 {
		return nil, &ValidationError{Field: "end_point", Message: "coordinates out of valid range"}
	}

	return &Trip{
		ID:         uuid.New(),
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: start,
		EndPoint:   end,
		Price:      price,
		Status:     StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}, nil
}

func Restore(
	id, userID, driverID uuid.UUID,
	start, end Coordinates,
	price int64,
	status Status,
	startedAt time.Time,
	finishedAt *time.Time,
	createdAt, updatedAt time.Time,
) *Trip {
	return &Trip{
		ID:         id,
		UserID:     userID,
		DriverID:   driverID,
		StartPoint: start,
		EndPoint:   end,
		Price:      price,
		Status:     status,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}
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
