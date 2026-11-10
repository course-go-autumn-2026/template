package trip

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

type Point struct {
	Latitude  float64
	Longitude float64
}

type Trip struct {
	Id             uuid.UUID
	UserId         uuid.UUID
	DriverId       uuid.UUID
	StartPoint     Point
	EndPoint       Point
	Price          int64
	Status         Status
	StartedAt      time.Time
	FinishedAt     *time.Time
	LastPositionAt *time.Time
}

func (t *Trip) Finish(at time.Time) error {
	if t.Status != StatusActive {
		return ErrTripCompleted
	}
	t.Status = StatusCompleted
	finished := at.UTC().Truncate(time.Microsecond)
	t.FinishedAt = &finished
	return nil
}
