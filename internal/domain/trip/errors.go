package trip

import (
	"errors"
	"fmt"
)

var (
	ErrIdempotencyConflict = errors.New("idempotency conflict")
	ErrInvalidRequest      = errors.New("invalid request")
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip already completed")
	ErrDriverBusy          = errors.New("driver already has an active trip")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}
