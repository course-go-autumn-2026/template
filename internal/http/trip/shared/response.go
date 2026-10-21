package shared

import (
	"encoding/json"
	"log/slog"
	"net/http"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	api "github.com/whxtelyy/template/internal/generated"
	"github.com/whxtelyy/template/internal/http/domainerror"
)

func ToAPITrip(trip domain.Trip) api.Trip {
	return api.Trip{
		Id:             trip.ID,
		UserId:         trip.UserID,
		DriverId:       trip.DriverID,
		StartPoint:     api.Coordinates{Latitude: trip.StartPoint.Latitude, Longitude: trip.StartPoint.Longitude},
		EndPoint:       api.Coordinates{Latitude: trip.EndPoint.Latitude, Longitude: trip.EndPoint.Longitude},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: trip.LastPositionAt,
	}
}

func WriteServiceError(w http.ResponseWriter, r *http.Request, err error, logger *slog.Logger) {
	status, problem := domainerror.Problem(err)
	if status == http.StatusInternalServerError {
		logger.ErrorContext(r.Context(), "request failed", "error", err.Error())
	}
	instance := r.URL.Path
	problem.Instance = &instance
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func WriteProblem(w http.ResponseWriter, r *http.Request, status int32, code, title, detail, problemType string) {
	instance := r.URL.Path
	problem := api.Problem{
		Type:     problemType,
		Title:    title,
		Status:   status,
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(status))
	_ = json.NewEncoder(w).Encode(problem)
}
