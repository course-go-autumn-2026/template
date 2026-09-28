package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	api "github.com/redukeee-hse/avitoService/internal/generated"
	"github.com/redukeee-hse/avitoService/internal/model"
)

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var req api.TripData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeInvalidRequest(w, r, "invalid JSON body")
		return
	}

	if req.UserId == uuid.Nil || req.DriverId == uuid.Nil {
		writeInvalidRequest(w, r, "user_id and driver_id are required")
		return
	}
	if !validPoint(req.StartPoint) || !validPoint(req.EndPoint) {
		writeInvalidRequest(w, r, "coordinates are out of range")
		return
	}
	if req.Price < 0 {
		writeInvalidRequest(w, r, "price must be >= 0")
		return
	}

	trip, err := h.Service.CreateTrip(r.Context(), model.Trip{
		UserID:     req.UserId,
		DriverID:   req.DriverId,
		StartPoint: model.Coordinates{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		EndPoint:   model.Coordinates{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:      req.Price,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID.String())
	writeJSON(w, http.StatusCreated, toAPITrip(trip))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.Service.GetTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.Service.FinishTrip(r.Context(), tripId)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func validPoint(c api.Coordinates) bool {
	return c.Latitude >= -90 && c.Latitude <= 90 &&
		c.Longitude >= -180 && c.Longitude <= 180
}

func toAPITrip(t model.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
