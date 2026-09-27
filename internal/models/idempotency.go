package models

import (
	"time"

	"github.com/google/uuid"
)

type IdempotencyRecord struct {
	Key         uuid.UUID
	RequestHash []byte
	TripID      *uuid.UUID
	ExpiresAt   time.Time
}
