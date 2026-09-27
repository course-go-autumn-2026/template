package domainerrors

import (
	"errors"
	"fmt"
	"net/http"

	httpres "github.com/imrtlDrifter/template/pkg/http_res"
)

const (
	InvalidRequestCode  = "invalid_request"
	TripNotFoundCode    = "trip_not_found"
	TripCompletedCode   = "trip_completed"
	DriverBusyCode      = "driver_busy"
	InvalidRequestTitle = "Invalid request"
	TripNotFoundTitle   = "Trip not found"
	TripCompletedTitle  = "Trip completed"
	DriverBusyTitle     = "Driver busy"
)

var (
	ErrEmptyRequestBody = httpres.NewHTTPError(
		errors.New("request body is required"),
		http.StatusBadRequest,
		InvalidRequestCode,
		InvalidRequestTitle,
		"Request body is required",
	)
	ErrTripNotFound = httpres.NewHTTPError(
		errors.New("trip not found"),
		http.StatusNotFound,
		TripNotFoundCode,
		TripNotFoundTitle,
		"Trip was not found",
	)
	ErrTripCompleted = httpres.NewHTTPError(
		errors.New("trip already completed"),
		http.StatusConflict,
		TripCompletedCode,
		TripCompletedTitle,
		"Operation is not allowed for a completed trip",
	)
	ErrDriverBusy = httpres.NewHTTPError(
		errors.New("driver already has an active trip"),
		http.StatusConflict,
		DriverBusyCode,
		DriverBusyTitle,
		"Driver already has an active trip",
	)
	ErrIdempotencyConflict = httpres.NewHTTPError(
		errors.New("idempotency key was used with another request"),
		http.StatusConflict,
		"idempotency_conflict",
		"Idempotency conflict",
		"Idempotency-Key was already used with a different request body",
	)
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %s: %s", e.Field, e.Message)
}

func WrapValidationError(err error) error {
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		return err
	}

	return httpres.NewHTTPError(err, http.StatusBadRequest, InvalidRequestCode, InvalidRequestTitle, validationErr.Error())
}
