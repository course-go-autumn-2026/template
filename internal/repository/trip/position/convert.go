package position

import "github.com/George1487/MalyginGeorge/internal/domain/trip"

type scannable interface {
	Scan(dest ...any) error
}

func scanPosition(row scannable) (trip.Position, error) {
	var p trip.Position
	err := row.Scan(
		&p.ID,
		&p.TripID,
		&p.Coordinates.Latitude,
		&p.Coordinates.Longitude,
		&p.RecordedAt,
		&p.CreatedAt,
	)
	return p, err
}
