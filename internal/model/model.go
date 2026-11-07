package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTripNotFound   = errors.New("trip not found")
	ErrTripCompleted  = errors.New("trip completed")
	ErrDriverBusy     = errors.New("driver busy")
	ErrInvalidRequest = errors.New("invalid request")
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartPoint     Point
	EndPoint       Point
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

type TripStatusHistory struct {
	ID        int64
	TripID    uuid.UUID
	Status    string
	ChangedAt time.Time
}
