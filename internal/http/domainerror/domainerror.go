package domainerror

import (
	"errors"

	"github.com/murmyauuu/template/api"
	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

func Problem(err error) (int, api.Problem) {
	var ve domaintrip.ValidationError
	if errors.As(err, &ve) {
		detail := ve.Reason
		return 400, api.Problem{
			Type:   "https://tripgo.example/problems/invalid-request",
			Title:  "Invalid request",
			Status: 400,
			Code:   "invalid_request",
			Detail: &detail,
		}
	}
	switch {
	case errors.Is(err, domaintrip.ErrTripNotFound):
		detail := "Trip was not found"
		return 404, api.Problem{
			Type:   "https://tripgo.example/problems/trip-not-found",
			Title:  "Trip not found",
			Status: 404,
			Code:   "trip_not_found",
			Detail: &detail,
		}
	case errors.Is(err, domaintrip.ErrTripCompleted):
		detail := "Operation is not allowed for a completed trip"
		return 409, api.Problem{
			Type:   "https://tripgo.example/problems/trip-completed",
			Title:  "Trip completed",
			Status: 409,
			Code:   "trip_completed",
			Detail: &detail,
		}
	case errors.Is(err, domaintrip.ErrDriverBusy):
		detail := "Driver already has an active trip"
		return 409, api.Problem{
			Type:   "https://tripgo.example/problems/driver-busy",
			Title:  "Driver busy",
			Status: 409,
			Code:   "driver_busy",
			Detail: &detail,
		}
	case errors.Is(err, domaintrip.ErrIdempotencyConflict):
		detail := "Idempotency key was already used with different request data"
		return 409, api.Problem{
			Type:   "https://tripgo.example/problems/idempotency-conflict",
			Title:  "Idempotency conflict",
			Status: 409,
			Code:   "idempotency_conflict",
			Detail: &detail,
		}
	default:
		detail := "Unable to process the request"
		return 500, api.Problem{
			Type:   "https://tripgo.example/problems/internal-error",
			Title:  "Internal server error",
			Status: 500,
			Code:   "internal_error",
			Detail: &detail,
		}
	}
}
