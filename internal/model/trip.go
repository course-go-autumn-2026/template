package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
)

type (
	Trip   = trip.Trip
	Point  = trip.Point
	Status = trip.Status
)

const (
	StatusActive    = trip.StatusActive
	StatusCompleted = trip.StatusCompleted
)

type TripStatusHistory struct {
	ID        int64
	TripID    uuid.UUID
	Status    string
	ChangedAt time.Time
}
