package domainerror

import (
	"errors"
	"net/http"

	"github.com/toooriii/tori/internal/domain/trip"
	api "github.com/toooriii/tori/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	var ve *trip.ValidationError
	switch {
	case errors.As(err, &ve):
		return http.StatusBadRequest, problem(http.StatusBadRequest, "invalid_request", ve.Error())
	case errors.Is(err, trip.ErrTripNotFound):
		return http.StatusNotFound, problem(http.StatusNotFound, "trip_not_found", "Trip was not found")
	case errors.Is(err, trip.ErrTripCompleted):
		return http.StatusConflict, problem(http.StatusConflict, "trip_completed", "Operation is not allowed for a completed trip")
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, problem(http.StatusConflict, "driver_busy", "Driver already has an active trip")
	case errors.Is(err, trip.ErrIdempotencyConflict):
		return http.StatusConflict, problem(http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request body")
	case errors.Is(err, trip.ErrKeyAlreadyExists):
		return http.StatusConflict, problem(http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used")
	default:
		return http.StatusInternalServerError, problem(http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func problem(status int, code, detail string) api.Problem {
	d := detail
	return api.Problem{
		Type:   "https://tripgo.example/problems/" + code,
		Title:  titleOf(code),
		Status: int32(status), //nolint:gosec
		Detail: &d,
		Code:   code,
	}
}

func titleOf(code string) string {
	switch code {
	case "invalid_request":
		return "Invalid request"
	case "trip_not_found":
		return "Trip not found"
	case "trip_completed":
		return "Trip completed"
	case "driver_busy":
		return "Driver busy"
	case "idempotency_conflict":
		return "Idempotency conflict"
	default:
		return "Internal Server Error"
	}
}
