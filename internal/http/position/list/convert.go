package list

import (
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/model"
)

func toAPIPositionList(positions []model.Position) api.TripPositionsResponse {
	items := make([]api.TripPosition, 0, len(positions))
	for _, p := range positions {
		items = append(items, api.TripPosition{
			Id:     p.ID,
			TripId: p.TripID,
			Coordinates: api.Coordinates{
				Latitude:  p.Coordinates.Latitude,
				Longitude: p.Coordinates.Longitude,
			},
			RecordedAt: p.RecordedAt,
		})
	}
	return api.TripPositionsResponse{Positions: items}
}
