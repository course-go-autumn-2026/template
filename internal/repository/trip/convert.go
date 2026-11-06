package trip

import (
	"github.com/jackc/pgx/v5"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/model"
)

func scanTrip(row pgx.Row) (model.Trip, error) {
	var (
		status string
		t      model.Trip
	)
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.Start.Latitude, &t.Start.Longitude, &t.End.Latitude, &t.End.Longitude,
		&t.Price, &status, &t.StartedAt, &t.FinishedAt,
	)
	if err != nil {
		return model.Trip{}, err
	}
	t.Status = trip.Status(status)
	t.StartedAt = t.StartedAt.UTC()
	if t.FinishedAt != nil {
		u := t.FinishedAt.UTC()
		t.FinishedAt = &u
	}
	return t, nil
}
