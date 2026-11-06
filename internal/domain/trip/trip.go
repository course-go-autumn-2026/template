package trip

import (
	"time"

	"github.com/google/uuid"
)

const msgEmptyUUID = "must be non-empty UUID"

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	DriverID   uuid.UUID
	Start      Point
	End        Point
	Price      int64
	Status     Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	t.FinishedAt = &at
	return nil
}

func (t Trip) Validate() error {
	if t.ID == uuid.Nil {
		return &ValidationError{Field: "id", Message: msgEmptyUUID}
	}
	if t.UserID == uuid.Nil {
		return &ValidationError{Field: "user_id", Message: msgEmptyUUID} // <-- ЗАМЕНИЛИ
	}
	if t.DriverID == uuid.Nil {
		return &ValidationError{Field: "driver_id", Message: msgEmptyUUID} // <-- ЗАМЕНИЛИ
	}
	if err := validatePoint("start_point", t.Start); err != nil {
		return err
	}
	if err := validatePoint("end_point", t.End); err != nil {
		return err
	}
	if t.Price < 0 {
		return &ValidationError{Field: "price", Message: "must be >= 0"}
	}
	if t.Status != StatusActive && t.Status != StatusCompleted {
		return &ValidationError{Field: "status", Message: "unknown status"}
	}
	return nil
}

func validatePoint(name string, p Point) error {
	if p.Latitude < -90 || p.Latitude > 90 {
		return &ValidationError{Field: name + ".latitude", Message: "must be in [-90, 90]"}
	}
	if p.Longitude < -180 || p.Longitude > 180 {
		return &ValidationError{Field: name + ".longitude", Message: "must be in [-180, 180]"}
	}
	return nil
}
