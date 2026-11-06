package model

import (
	"time"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/domain/trip"
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
	Status     trip.Status
	StartedAt  time.Time
	FinishedAt *time.Time
}

func FromDomain(t trip.Trip) Trip {
	return Trip{
		ID:         t.ID,
		UserID:     t.UserID,
		DriverID:   t.DriverID,
		Start:      Point{Latitude: t.Start.Latitude, Longitude: t.Start.Longitude},
		End:        Point{Latitude: t.End.Latitude, Longitude: t.End.Longitude},
		Price:      t.Price,
		Status:     t.Status,
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}

func (m Trip) ToDomain() trip.Trip {
	return trip.Trip{
		ID:         m.ID,
		UserID:     m.UserID,
		DriverID:   m.DriverID,
		Start:      trip.Point{Latitude: m.Start.Latitude, Longitude: m.Start.Longitude},
		End:        trip.Point{Latitude: m.End.Latitude, Longitude: m.End.Longitude},
		Price:      m.Price,
		Status:     m.Status,
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
	}
}
