package create

import (
	"github.com/google/uuid"

	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/model"
	ucposcreate "github.com/xotapy-main/courseGo/internal/usecase/position/create"
)

func toUsecaseRequest(tripID api.TripId, req api.CreateTripPositionJSONRequestBody) ucposcreate.Request {
	return ucposcreate.Request{
		TripID: uuid.UUID(tripID),
		Coordinates: model.Point{
			Latitude:  req.Coordinates.Latitude,
			Longitude: req.Coordinates.Longitude,
		},
		RecordedAt: req.RecordedAt,
	}
}

func toAPIPosition(p *model.Position) api.TripPosition {
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
