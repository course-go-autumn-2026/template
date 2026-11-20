package trip

import (
	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type scannable interface {
	Scan(dest ...any) error
}

func scanTrip(row scannable) (trip.Trip, error) {
	var t trip.Trip
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.DriverID,
		&t.StartPoint.Latitude,
		&t.StartPoint.Longitude,
		&t.EndPoint.Latitude,
		&t.EndPoint.Longitude,
		&t.Price,
		&t.Status,
		&t.StartedAt,
		&t.FinishedAt,
		&t.LastPositionAt,
	)
	return t, err
}
