package create

import (
	"github.com/murmyauuu/template/api"
	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	uc "github.com/murmyauuu/template/internal/usecase/trip/create"
)

func toInput(data api.TripData) uc.Input {
	return uc.Input{
		UserId:     data.UserId,
		DriverId:   data.DriverId,
		StartPoint: domaintrip.Point{Latitude: data.StartPoint.Latitude, Longitude: data.StartPoint.Longitude},
		EndPoint:   domaintrip.Point{Latitude: data.EndPoint.Latitude, Longitude: data.EndPoint.Longitude},
		Price:      data.Price,
	}
}

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
