package convert

import (
	"github.com/freyliee/template/internal/domain/trip"
	api "github.com/freyliee/template/internal/generated"
)

func Position(p trip.Position) api.TripPosition {
	return api.TripPosition{
		Id:          p.ID,
		TripId:      p.TripID,
		Coordinates: api.Coordinates{Latitude: p.Coordinates.Latitude, Longitude: p.Coordinates.Longitude},
		RecordedAt:  p.RecordedAt,
	}
}
