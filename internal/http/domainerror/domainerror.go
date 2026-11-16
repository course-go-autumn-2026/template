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
		detail := valErr.Error()
		return http.StatusBadRequest, api.Problem{
			Type:     "https://tripgo.example/problems/invalid-request",
			Title:    "Invalid request",
			Status:   int32(http.StatusBadRequest),
			Detail:   &detail,
			Instance: &instance,
			Code:     "invalid_request",
		}
	}

	switch {
	case errors.Is(err, trip.ErrTripNotFound):
		detail := "Trip not found"
		return http.StatusNotFound, api.Problem{
			Type:     "https://tripgo.example/problems/trip-not-found",
			Title:    "Trip not found",
			Status:   int32(http.StatusNotFound),
			Detail:   &detail,
			Instance: &instance,
			Code:     "trip_not_found",
		}
	case errors.Is(err, trip.ErrTripCompleted):
		detail := "Trip is already completed"
		return http.StatusConflict, api.Problem{
			Type:     "https://tripgo.example/problems/trip-completed",
			Title:    "Trip completed",
			Status:   int32(http.StatusConflict),
			Detail:   &detail,
			Instance: &instance,
			Code:     "trip_completed",
		}
	case errors.Is(err, trip.ErrDriverBusy):
		detail := "Driver already has an active trip"
		return http.StatusConflict, api.Problem{
			Type:     "https://tripgo.example/problems/driver-busy",
			Title:    "Driver busy",
			Status:   int32(http.StatusConflict),
			Detail:   &detail,
			Instance: &instance,
			Code:     "driver_busy",
		}
	default:
		detail := "Internal server error"
		return http.StatusInternalServerError, api.Problem{
			Type:     "https://tripgo.example/problems/internal-error",
			Title:    "Internal server error",
			Status:   int32(http.StatusInternalServerError),
			Detail:   &detail,
			Instance: &instance,
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
