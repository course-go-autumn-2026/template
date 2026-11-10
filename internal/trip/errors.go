package trip

import "errors"

var (
	ErrDriverBusy          = errors.New("driver already has an active trip")
	ErrTripNotFound        = errors.New("trip not found")
	ErrTripCompleted       = errors.New("trip already completed")
	ErrIdempotencyConflict = errors.New("idempotency key already used with different request data")
)
