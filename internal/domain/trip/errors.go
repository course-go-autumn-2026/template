package trip

import (
	"errors"
	"fmt"
)

var (
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip already completed")
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrIdempotencyConflict = errors.New("idempotency key reused with different body")
	ErrKeyAlreadyExists    = errors.New("idempotency key already exists")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %s %s", e.Field, e.Message)
}
