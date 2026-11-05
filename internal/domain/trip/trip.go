package trip

import (
	"time"
	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusCompleted Status = "completed"
)

func FinishTrip(t *appmodel.Trip, at time.Time) error {
	if t.Status != string(StatusActive) {
		return ErrTripCompleted
	}

	t.Status = string(StatusCompleted) 

	t.FinishedAt = &at 
	return nil
}