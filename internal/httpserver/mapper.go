package httpserver

import (
	api "github.com/Yolanesso/avito-labs/internal/generated"
	"github.com/Yolanesso/avito-labs/internal/model"
)

func tripToAPI(trip model.Trip) api.Trip {
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

		Price:  trip.Price,
		Status: api.TripStatus(trip.Status),

		StartedAt:  trip.StartedAt,
		FinishedAt: trip.FinishedAt,

		LastPositionAt: nil,
	}
}
