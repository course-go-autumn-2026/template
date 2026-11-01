package trips

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Kind uint8

const (
	KindNotFound Kind = iota + 1
	KindConflict
)

type Error struct {
	Kind    Kind
	Code    string
	Message string
}

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	TripID   uuid.UUID
	UserID   uuid.UUID
	DriverID uuid.UUID

	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
	Status     Status

	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (e *Error) Error() string {
	return e.Message
}

var (
	ErrTripNotFound = &Error{
		Kind:    KindNotFound,
		Code:    "trip_not_found",
		Message: "Trip was not found",
	}
	ErrTripCompleted = &Error{
		Kind:    KindConflict,
		Code:    "trip_completed",
		Message: "Trip is already completed",
	}
	ErrDriverBusy = &Error{
		Kind:    KindConflict,
		Code:    "driver_busy",
		Message: "Driver already has an active trip",
	}
)
