package models

import (
	"time"

	"github.com/google/uuid"
	api "github.com/imrtlDrifter/template/internal/generated"
)

type TripStatus string

const (
	TripStatusActive    TripStatus = "active"
	TripStatusCompleted TripStatus = "completed"
)

type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type CreateTripCommand struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint Coordinates
	EndPoint   Coordinates
	Price      int64
}

type Trip struct {
	ID             uuid.UUID   `json:"id"`
	UserID         uuid.UUID   `json:"user_id"`
	DriverID       uuid.UUID   `json:"driver_id"`
	StartPoint     Coordinates `json:"start_point"`
	EndPoint       Coordinates `json:"end_point"`
	Price          int64       `json:"price"`
	Status         TripStatus  `json:"status"`
	StartedAt      time.Time   `json:"started_at"`
	FinishedAt     *time.Time  `json:"finished_at,omitempty"`
	LastPositionAt *time.Time  `json:"last_position_at,omitempty"`
}

type TripStatusHistory struct {
	ID         int64       `json:"id"`
	TripID     uuid.UUID   `json:"trip_id"`
	FromStatus *TripStatus `json:"from_status,omitempty"`
	ToStatus   TripStatus  `json:"to_status"`
	Reason     *string     `json:"reason,omitempty"`
	ChangedAt  time.Time   `json:"changed_at"`
}

func TripResponse(trip *Trip) api.Trip {
	return api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  trip.StartPoint.Latitude,
			Longitude: trip.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndPoint.Latitude,
			Longitude: trip.EndPoint.Longitude,
		},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: trip.LastPositionAt,
	}
}

func CreateTrip(request api.CreateTripJSONRequestBody) *CreateTripCommand {
	return &CreateTripCommand{
		UserID:   request.UserId,
		DriverID: request.DriverId,
		StartPoint: Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},
		EndPoint: Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},
		Price: request.Price,
	}
}
