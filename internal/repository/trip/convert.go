package trip

import (
	"time"

	"github.com/google/uuid"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
)

type dbTrip struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	DriverID       uuid.UUID
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Price          int64
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (d *dbTrip) toModel() *model.Trip {
	return &model.Trip{
		ID:       d.ID,
		UserID:   d.UserID,
		DriverID: d.DriverID,
		StartPoint: trip.Coordinates{
			Latitude:  d.StartLatitude,
			Longitude: d.StartLongitude,
		},
		EndPoint: trip.Coordinates{
			Latitude:  d.EndLatitude,
			Longitude: d.EndLongitude,
		},
		Price:      d.Price,
		Status:     trip.Status(d.Status),
		StartedAt:  d.StartedAt,
		FinishedAt: d.FinishedAt,
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt,
	}
}
