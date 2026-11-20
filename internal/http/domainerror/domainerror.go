// Package domainerror maps domain errors to RFC 9457 problem responses.
package domainerror

import (
	"errors"
	"net/http"
	"strings"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
)

// Problem maps a domain error to an HTTP status and RFC 9457 problem body.
func Problem(err error) (int, api.Problem) {
	var validationErr *trip.ValidationError

	switch {
	case errors.Is(err, trip.ErrTripNotFound):
		return problem(http.StatusNotFound, "trip_not_found", "Trip not found", "Trip not found")
	case errors.Is(err, trip.ErrTripCompleted):
		return problem(http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
	case errors.Is(err, trip.ErrDriverBusy):
		return problem(http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, trip.ErrConflict):
		return problem(http.StatusConflict, "conflict", "Conflict", "Trip conflicts with existing data")
	case errors.Is(err, trip.ErrIdempotencyConflict):
		return problem(http.StatusConflict, "idempotency_conflict", "Idempotency conflict", "Idempotency key was used with a different request")
	case errors.As(err, &validationErr):
		return problem(http.StatusBadRequest, "invalid_request", "Invalid request", validationErr.Error())
	default:
		return problem(http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred")
	}
}

func problem(status int, code, title, detail string) (int, api.Problem) {
	return status, api.Problem{
		Type:   "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:  title,
		Status: int32(status), //nolint:gosec // status is always one of the fixed HTTP codes used in this file
		Detail: &detail,
		Code:   code,
	}
}
