package trip

import (
	"github.com/jackc/pgx/v5"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
)

func scanTrip(row pgx.Row) (*model.Trip, error) {
	var (
		t      model.Trip
		status string
	)
	err := row.Scan(
		&t.ID, &t.UserID, &t.DriverID,
		&t.StartPoint.Latitude, &t.StartPoint.Longitude,
		&t.EndPoint.Latitude, &t.EndPoint.Longitude,
		&t.Price, &status, &t.StartedAt,
		&t.FinishedAt, &t.LastPositionAt,
	)
	if err != nil {
		return nil, err
	}
	t.Status = domaintrip.Status(status)
	return &t, nil
}
