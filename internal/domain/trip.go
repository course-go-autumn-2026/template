package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTripNotFound   = errors.New("trip not found")
	ErrDriverBusy     = errors.New("driver already has an active trip")
	ErrTripCompleted  = errors.New("trip is already completed")
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