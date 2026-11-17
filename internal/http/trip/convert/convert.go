package convert

import (
	"github.com/freyliee/template/internal/domain/trip"
	api "github.com/freyliee/template/internal/generated"
)

func Data(d api.CreateTripJSONRequestBody) trip.Data {
	return trip.Data{UserID: d.UserId, DriverID: d.DriverId, StartPoint: trip.Coordinates{Latitude: d.StartPoint.Latitude, Longitude: d.StartPoint.Longitude}, EndPoint: trip.Coordinates{Latitude: d.EndPoint.Latitude, Longitude: d.EndPoint.Longitude}, Price: d.Price}
}

func Response(t trip.Trip) api.Trip {
	return api.Trip{Id: t.ID, UserId: t.UserID, DriverId: t.DriverID, StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude}, EndPoint: api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude}, Price: t.Price, Status: api.TripStatus(t.Status), StartedAt: t.StartedAt, FinishedAt: t.FinishedAt, LastPositionAt: t.LastPositionAt}
}
