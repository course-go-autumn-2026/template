package finish

import (
	"github.com/murmyauuu/template/api"
	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

func toAPI(t domaintrip.Trip) api.Trip {
	return api.Trip{
		Id:         t.Id,
		UserId:     t.UserId,
		DriverId:   t.DriverId,
		StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
