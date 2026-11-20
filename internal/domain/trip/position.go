package trip

import (
	"time"

	"github.com/google/uuid"
)

type Position struct {
	ID          int64
	TripID      uuid.UUID
	Coordinates Coordinates
	CreatedAt   *time.Time
	RecordedAt  time.Time
}
