package create

import (
	"github.com/google/uuid"

	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/model"
	uccreate "github.com/xotapy-main/courseGo/internal/usecase/trip/create"
)

func toUsecaseRequest(req api.CreateTripRequest) uccreate.Request {
	return uccreate.Request{
		UserID:     uuid.UUID(req.UserId),
		DriverID:   uuid.UUID(req.DriverId),
		StartPoint: model.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		EndPoint:   model.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:      req.Price,
	}
}

func ToAPITrip(t *model.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Point{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:       api.Point{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}
