package create

import (
	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/model"
)

func toCreateTripData(req api.TripData) model.CreateTripData {
	return model.CreateTripData{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		StartPoint: trip.Coordinates{
			Latitude:  req.StartPoint.Latitude,
			Longitude: req.StartPoint.Longitude,
		},
		EndPoint: trip.Coordinates{
			Latitude:  req.EndPoint.Latitude,
			Longitude: req.EndPoint.Longitude,
		},
		Price: req.Price,
	}
}

func toResponse(t trip.Trip) api.Trip {
	return api.Trip{
		Id:       t.ID,
		UserId:   t.UserID,
		DriverId: t.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}
