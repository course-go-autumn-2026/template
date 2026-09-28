package trip_status_history

import (
	"time"

	"github.com/google/uuid"
	"github.com/makkssik/template/internal/domain/trips"
)

type StatusChange struct {
	TripID     uuid.UUID
	FromStatus *trips.Status
	ToStatus   trips.Status
	Reason     *string
	ChangedAt  time.Time
}
