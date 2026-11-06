package model

import "github.com/google/uuid"

type IdempotencyRecord struct {
	Key         uuid.UUID
	RequestHash string
	TripID      uuid.UUID
}
