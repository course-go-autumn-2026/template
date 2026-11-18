package create

import (
	"github.com/XrystikLK/template/internal/model"
	"github.com/XrystikLK/template/internal/usecase/trip/create"
)

func toUseCaseRequestFromRaw(req rawCreateTripRequest) create.Request {
	return create.Request{
		UserID:   *req.UserID,
		DriverID: *req.DriverID,
		StartPoint: model.Coordinates{
			Latitude:  *req.StartPoint.Latitude,
			Longitude: *req.StartPoint.Longitude,
		},
		EndPoint: model.Coordinates{
			Latitude:  *req.EndPoint.Latitude,
			Longitude: *req.EndPoint.Longitude,
		},
		Price: *req.Price,
	}
}
