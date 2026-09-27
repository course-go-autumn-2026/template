package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/imrtlDrifter/template/internal/domainerrors"
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

type RequestCoordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type CreateTripRequest struct {
	UserID     uuid.UUID           `json:"user_id"`
	DriverID   uuid.UUID           `json:"driver_id"`
	StartPoint *RequestCoordinates `json:"start_point"`
	EndPoint   *RequestCoordinates `json:"end_point"`
	Price      *int64              `json:"price"`
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

func (r *CreateTripRequest) Validate() error {
	switch {
	case r.UserID == uuid.Nil:
		return &domainerrors.ValidationError{
			Field:   "user_id",
			Message: "must be a non-empty UUID",
		}
	case r.DriverID == uuid.Nil:
		return &domainerrors.ValidationError{
			Field:   "driver_id",
			Message: "must be a non-empty UUID",
		}
	case r.StartPoint == nil:
		return &domainerrors.ValidationError{
			Field:   "start_point",
			Message: "is required",
		}
	case r.StartPoint.Latitude == nil:
		return &domainerrors.ValidationError{
			Field:   "start_point.latitude",
			Message: "is required",
		}
	case *r.StartPoint.Latitude < -90 || *r.StartPoint.Latitude > 90:
		return &domainerrors.ValidationError{
			Field:   "start_point.latitude",
			Message: "must be between -90 and 90",
		}
	case r.StartPoint.Longitude == nil:
		return &domainerrors.ValidationError{
			Field:   "start_point.longitude",
			Message: "is required",
		}
	case *r.StartPoint.Longitude < -180 || *r.StartPoint.Longitude > 180:
		return &domainerrors.ValidationError{
			Field:   "start_point.longitude",
			Message: "must be between -180 and 180",
		}
	case r.EndPoint == nil:
		return &domainerrors.ValidationError{
			Field:   "end_point",
			Message: "is required",
		}
	case r.EndPoint.Latitude == nil:
		return &domainerrors.ValidationError{
			Field:   "end_point.latitude",
			Message: "is required",
		}
	case *r.EndPoint.Latitude < -90 || *r.EndPoint.Latitude > 90:
		return &domainerrors.ValidationError{
			Field:   "end_point.latitude",
			Message: "must be between -90 and 90",
		}
	case r.EndPoint.Longitude == nil:
		return &domainerrors.ValidationError{
			Field:   "end_point.longitude",
			Message: "is required",
		}
	case *r.EndPoint.Longitude < -180 || *r.EndPoint.Longitude > 180:
		return &domainerrors.ValidationError{
			Field:   "end_point.longitude",
			Message: "must be between -180 and 180",
		}
	case r.Price == nil:
		return &domainerrors.ValidationError{
			Field:   "price",
			Message: "is required",
		}
	case *r.Price < 0:
		return &domainerrors.ValidationError{
			Field:   "price",
			Message: "must be greater than or equal to 0",
		}
	default:
		return nil
	}
}
