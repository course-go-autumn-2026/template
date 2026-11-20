package get

import (
	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

func toResponse(positions []trip.Position) api.TripPositionsResponse {
	result := make([]api.TripPosition, 0, len(positions))
	for _, p := range positions {
		result = append(result, api.TripPosition{
			Id:     p.ID,
			TripId: p.TripID,
			Coordinates: api.Coordinates{
				Latitude:  p.Coordinates.Latitude,
				Longitude: p.Coordinates.Longitude,
			},
			RecordedAt: p.RecordedAt,
		})
	}

	return api.TripPositionsResponse{
		Positions: result,
	}
}
