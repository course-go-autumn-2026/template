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
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Trip struct {
	Id         uuid.UUID  `json:"id"`
	UserId     uuid.UUID  `json:"user_id"`
	DriverId   uuid.UUID  `json:"driver_id"`
	StartPoint Point      `json:"start_point"`
	EndPoint   Point      `json:"end_point"`
	Price      int64      `json:"price"`
	Status     Status     `json:"status"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
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
