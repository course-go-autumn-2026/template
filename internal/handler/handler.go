package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5" // <-- Добавлен импорт chi
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	api "trip-service/internal/generated" // <-- Явный алиас api
	"trip-service/internal/repository"
	"trip-service/internal/service"
)

type Handler struct {
	service *service.TripService
	pool    *pgxpool.Pool
}

func New(service *service.TripService, pool *pgxpool.Pool) *Handler {
	return &Handler{
		service: service,
		pool:    pool,
	}
}

// POST /api/v1/trips
func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripJSONRequestBody
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	if req.StartPoint.Latitude < -90 || req.StartPoint.Latitude > 90 ||
		req.StartPoint.Longitude < -180 || req.StartPoint.Longitude > 180 ||
		req.EndPoint.Latitude < -90 || req.EndPoint.Latitude > 90 ||
		req.EndPoint.Longitude < -180 || req.EndPoint.Longitude > 180 ||
		req.Price < 0 {
		WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid coordinates or price", "Coordinates or price outside valid range")
		return
	}

	dto := service.CreateTripDTO{
		UserID:         req.UserId,
		DriverID:       req.DriverId,
		StartLatitude:  req.StartPoint.Latitude,
		StartLongitude: req.StartPoint.Longitude,
		EndLatitude:    req.EndPoint.Latitude,
		EndLongitude:   req.EndPoint.Longitude,
		Price:          req.Price,
	}

	trip, err := h.service.CreateTrip(r.Context(), dto)
	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			WriteProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Failed to create trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", trip.ID.String()))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(mapTripToResponse(trip))
}

// GET /api/v1/trips/{tripId}
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request) {
	tripIDStr := chi.URLParam(r, "tripId")
	tripID, err := uuid.Parse(tripIDStr)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid tripId", "tripId must be a valid UUID")
		return
	}

	trip, err := h.service.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			WriteProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip with given ID was not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Failed to fetch trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToResponse(trip))
}

// POST /api/v1/trips/{tripId}/finish
func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request) {
	tripIDStr := chi.URLParam(r, "tripId")
	tripID, err := uuid.Parse(tripIDStr)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid tripId", "tripId must be a valid UUID")
		return
	}

	trip, err := h.service.FinishTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			WriteProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip with given ID was not found")
			return
		}
		if errors.Is(err, repository.ErrTripCompleted) {
			WriteProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Failed to finish trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToResponse(trip))
}

// GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "pass"})
}

// GET /ready
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := h.pool.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "fail"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "pass"})
}

func mapTripToResponse(t *repository.Trip) map[string]any {
	return map[string]any{
		"id":        t.ID.String(),
		"user_id":   t.UserID.String(),
		"driver_id": t.DriverID.String(),
		"start_point": map[string]float64{
			"latitude":  t.StartLatitude,
			"longitude": t.StartLongitude,
		},
		"end_point": map[string]float64{
			"latitude":  t.EndLatitude,
			"longitude": t.EndLongitude,
		},
		"price":            t.Price,
		"status":           t.Status,
		"started_at":       t.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		"finished_at":      t.FinishedAt,
		"last_position_at": t.LastPositionAt,
	}
}
