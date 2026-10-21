package domainerror

import (
	"errors"
	"net/http"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	api "github.com/whxtelyy/template/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	status, code, title, detail, slug := http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error", "internal-error"
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		status, code, title, detail, slug = http.StatusBadRequest, "invalid_request", "Invalid request", validation.Error(), "invalid-request"
	case errors.Is(err, domain.ErrTripNotFound):
		status, code, title, detail, slug = http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found", "trip-not-found"
	case errors.Is(err, domain.ErrTripCompleted):
		status, code, title, detail, slug = http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip", "trip-completed"
	case errors.Is(err, domain.ErrDriverBusy):
		status, code, title, detail, slug = http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip", "driver-busy"
	case errors.Is(err, domain.ErrIdempotencyConflict):
		status, code, title, detail, slug = http.StatusConflict, "idempotency_conflict", "Idempotency Conflict", "Idempotency-Key was already used with a different request body", "idempotency-conflict"
	}
	return status, api.Problem{Type: "https://tripgo.example/problems/" + slug, Title: title, Status: int32(status), Detail: &detail, Code: code}
}
