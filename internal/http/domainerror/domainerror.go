package domainerror

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	api "github.com/xotapy-main/courseGo/internal/generated"
)

func Problem(err error, instance string) (int, api.Problem) {
	var valErr *trip.ValidationError
	if errors.As(err, &valErr) {
		return http.StatusBadRequest, api.Problem{
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   http.StatusBadRequest,
			Detail:   valErr.Error(),
			Instance: instance,
			Code:     "invalid_request",
		}
	}

	switch {
	case errors.Is(err, trip.ErrTripNotFound):
		return http.StatusNotFound, api.Problem{
			Type:     "https://tripgo.example/problems/trip-not-found",
			Title:    "Trip not found",
			Status:   http.StatusNotFound,
			Detail:   "Trip not found",
			Instance: instance,
			Code:     "trip_not_found",
		}
	case errors.Is(err, trip.ErrTripCompleted):
		return http.StatusConflict, api.Problem{
			Type:     "https://tripgo.example/problems/trip-completed",
			Title:    "Trip completed",
			Status:   http.StatusConflict,
			Detail:   "Trip is already completed",
			Instance: instance,
			Code:     "trip_completed",
		}
	case errors.Is(err, trip.ErrDriverBusy):
		return http.StatusConflict, api.Problem{
			Type:     "https://tripgo.example/problems/driver-busy",
			Title:    "Driver busy",
			Status:   http.StatusConflict,
			Detail:   "Driver already has an active trip",
			Instance: instance,
			Code:     "driver_busy",
		}
	default:
		return http.StatusInternalServerError, api.Problem{
			Type:     "https://tripgo.example/problems/internal-error",
			Title:    "Internal server error",
			Status:   http.StatusInternalServerError,
			Detail:   "Internal server error",
			Instance: instance,
			Code:     "internal_error",
		}
	}
}

func Write(w http.ResponseWriter, r *http.Request, err error) {
	status, prob := Problem(err, r.URL.Path)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(prob)
}
