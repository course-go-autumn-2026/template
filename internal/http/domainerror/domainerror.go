package domainerror

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/eternal-despair/template/internal/domain/trip"
	api "github.com/eternal-despair/template/internal/generated"
)

const problemType = "about:blank"

func Problem(err error) (int, api.Problem) {
	if err == nil {
		return http.StatusOK, api.Problem{}
	}

	var valErr *trip.ValidationError
	switch {
	case errors.As(err, &valErr):
		detail := valErr.Error()
		return http.StatusBadRequest, api.Problem{
			Type:   problemType,
			Title:  "Invalid request",
			Status: http.StatusBadRequest,
			Code:   "invalid_request",
			Detail: &detail,
		}

	case errors.Is(err, trip.ErrDriverBusy):
		detail := "Driver already has an active trip"
		return http.StatusConflict, api.Problem{
			Type:   problemType,
			Title:  "Driver busy",
			Status: http.StatusConflict,
			Code:   "driver_busy",
			Detail: &detail,
		}

	case errors.Is(err, trip.ErrTripCompleted):
		detail := "Trip already completed"
		return http.StatusConflict, api.Problem{
			Type:   problemType,
			Title:  "Trip completed",
			Status: http.StatusConflict,
			Code:   "trip_completed",
			Detail: &detail,
		}

	case errors.Is(err, trip.ErrTripNotFound):
		detail := "Trip with specified ID was not found"
		return http.StatusNotFound, api.Problem{
			Type:   problemType,
			Title:  "Trip not found",
			Status: http.StatusNotFound,
			Code:   "trip_not_found",
			Detail: &detail,
		}

	default:
		detail := "Internal server error"
		return http.StatusInternalServerError, api.Problem{
			Type:   problemType,
			Title:  "Internal server error",
			Status: http.StatusInternalServerError,
			Code:   "internal_error",
			Detail: &detail,
		}
	}
}

func Write(w http.ResponseWriter, err error) {
	status, prob := Problem(err)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}

func WriteWithDetail(w http.ResponseWriter, status int, code, title, detail string) {
	prob := api.Problem{
		Type:   problemType,
		Title:  title,
		Status: int32(status), //nolint:gosec // HTTP status code is always between 100 and 599
		Code:   code,
		Detail: &detail,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
