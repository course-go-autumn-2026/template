package domainerror

import (
	"errors"
	"net/http"

	"github.com/ggwpfu9/courseAvito/internal/domain/trip"
	api "github.com/ggwpfu9/courseAvito/internal/generated"
)

func Problem(err error) (int, api.Problem) {
	if errors.Is(err, trip.ErrTripNotFound) {
		return http.StatusNotFound, api.Problem{Code: "trip_not_found", Title: "Trip not found", Status: 404}
	}
	if errors.Is(err, trip.ErrTripCompleted) {
		return http.StatusConflict, api.Problem{Code: "trip_completed", Title: "Trip completed", Status: 409}
	}
	if errors.Is(err, trip.ErrDriverBusy) {
		return http.StatusConflict, api.Problem{Code: "driver_busy", Title: "Driver busy", Status: 409}
	}

	return http.StatusInternalServerError, api.Problem{Code: "internal_error", Title: "Internal server error", Status: 500}
}