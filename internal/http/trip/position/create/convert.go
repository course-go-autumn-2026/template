package create

import (
	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/model"
)

func toCreatePosition(tripID uuid.UUID, req api.PositionData) model.CreatePosition {
	return model.CreatePosition{
		TripID:     tripID,
		Latitude:   req.Coordinates.Latitude,
		Longitude:  req.Coordinates.Longitude,
		RecordedAt: req.RecordedAt,
	}
}

func toResponse(p *trip.Position) api.TripPosition {
	return api.TripPosition{
		Id:     p.ID,
		TripId: p.TripID,
		Coordinates: api.Coordinates{
			Latitude:  p.Coordinates.Latitude,
			Longitude: p.Coordinates.Longitude,
		},
		RecordedAt: p.RecordedAt,
	}
}
