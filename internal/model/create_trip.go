package model

import (
	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type CreateTripData struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint trip.Coordinates
	EndPoint   trip.Coordinates
	Price      int64
}
