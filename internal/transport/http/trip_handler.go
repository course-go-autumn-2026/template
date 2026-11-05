package http

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/ggwpfu9/courseAvito/internal/domain"
	api "github.com/ggwpfu9/courseAvito/internal/generated"
	"github.com/ggwpfu9/courseAvito/internal/service"
)

type TripHandler struct {
	svc *service.TripService
}

func NewTripHandler(svc *service.TripService) *TripHandler {
	return &TripHandler{svc: svc}
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body", r.URL.Path)
		return
	}

	if req.Price < 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "Price must be >= 0", r.URL.Path)
		return
	}

	trip := &domain.Trip{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		StartPoint: domain.Point{
			Latitude:  req.StartPoint.Latitude,
			Longitude: req.StartPoint.Longitude,
		},
		EndPoint: domain.Point{
			Latitude:  req.EndPoint.Latitude,
			Longitude: req.EndPoint.Longitude,
		},
		Price: int64(req.Price),
	}

	if err := h.svc.CreateTrip(r.Context(), trip); err != nil {
		if err == domain.ErrDriverBusy {
			writeError(w, http.StatusConflict, "driver_busy", "Driver already has an active trip", r.URL.Path)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to create trip", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tripToResponse(trip))
}

func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId string) {
	id, err := uuid.Parse(tripId)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid UUID format", r.URL.Path)
		return
	}

	trip, err := h.svc.GetTrip(r.Context(), id)
	if err != nil {
		if err == domain.ErrTripNotFound {
			writeError(w, http.StatusNotFound, "trip_not_found", "Trip not found", r.URL.Path)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to get trip", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tripToResponse(trip))
}

func tripToResponse(t *domain.Trip) api.Trip {
	return api.Trip{
		Id:        t.ID,
		UserId:    t.UserID,
		DriverId:  t.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}

func writeError(w http.ResponseWriter, status int, code, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":     "https://tripgo.example/problems/" + code,
		"title":    http.StatusText(status),
		"status":   status,
		"detail":   detail,
		"code":     code,
		"instance": instance,
	})
}

func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.svc.FinishTrip(r.Context(), tripId)
	if err != nil {
		if err == domain.ErrTripNotFound {
			writeError(w, http.StatusNotFound, "trip_not_found", "Trip not found", r.URL.Path)
			return
		}
		if err == domain.ErrTripCompleted {
			writeError(w, http.StatusConflict, "trip_completed", "Trip is already completed", r.URL.Path)
			return
		}
		writeError(w, http.StatusInternalServerError, "internal_error", "Failed to finish trip", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tripToResponse(trip))
}