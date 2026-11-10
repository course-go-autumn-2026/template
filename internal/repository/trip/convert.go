package trip

import (
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

func scanTrip(row pgx.Row) (domaintrip.Trip, error) {
	var value domaintrip.Trip
	var status string
	err := row.Scan(&value.Id, &value.UserId, &value.DriverId,
		&value.StartPoint.Latitude, &value.StartPoint.Longitude,
		&value.EndPoint.Latitude, &value.EndPoint.Longitude,
		&value.Price, &status, &value.StartedAt, &value.FinishedAt)
	if err != nil {
		return domaintrip.Trip{}, err
	}
	value.Status = domaintrip.Status(status)
	value.StartedAt = value.StartedAt.UTC()
	if value.FinishedAt != nil {
		finishedAt := value.FinishedAt.UTC()
		value.FinishedAt = &finishedAt
	}
	return value, nil
}
