package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"encoding/json" 

	api "github.com/Hil-sys/TripGo-autumn/internal/api"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"
)

// RFC 9457
type Problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
	Code     string `json:"code"`
}

type TripHandler struct {
	repo      *repository.TripRepository
	txManager *repository.TxManager
}

func NewTripHandler(repo *repository.TripRepository, txManager *repository.TxManager) *TripHandler {
	return &TripHandler{
		repo:      repo,
		txManager: txManager,
	}
}

// Вспомогательный метод
func (h *TripHandler) respondWithError(w http.ResponseWriter, r *http.Request, httpStatus int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(httpStatus)
	
	prob := Problem{
		Type:     "https://tripgo.example" + code,
		Title:    title,
		Status:   httpStatus,
		Detail:   detail,
		Instance: r.URL.Path,
		Code:     code,
	}
	_ = json.NewEncoder(w).Encode(prob)
}

func isInvalidUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err != nil
}

// 1. POST /api/v1/trips - Создание поездки
func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var req struct {
		UserID     string `json:"user_id"`
		DriverID   string `json:"driver_id"`
		StartPoint struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"start_point"`
		EndPoint struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"end_point"`
		Price int64 `json:"price"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondWithError(w, r, http.StatusBadRequest, "invalid_request", "Bad Request", "Invalid JSON body")
		return
	}

	if isInvalidUUID(req.UserID) || isInvalidUUID(req.DriverID) || req.Price < 0 {
		h.respondWithError(w, r, http.StatusBadRequest, "invalid_request", "Bad Request", "Validation failed")
		return
	}

	tripID := uuid.New().String()
	
	trip := &repository.Trip{
		ID:         tripID,
		UserID:     req.UserID,
		DriverID:   req.DriverID,
		StartPoint: repository.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		EndPoint:   repository.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:      req.Price,
		Status:     "active",
	}

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		return h.repo.CreateTrip(ctx, trip)
	})

	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			h.respondWithError(w, r, http.StatusConflict, "driver_busy", "Driver Busy", "Driver already has an active trip")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", "/api/v1/trips/"+tripID)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(trip)
}

// 2. GET /api/v1/trips/{tripId} - Получение поездки
func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	tripStr := tripId.String()

	trip, err := h.repo.GetTrip(r.Context(), tripStr)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			h.respondWithError(w, r, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}

// 3. POST /api/v1/trips/{tripId}/finish - Завершение поездки
func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	tripStr := tripId.String()

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		return h.repo.FinishTrip(ctx, tripStr)
	})

	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			h.respondWithError(w, r, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		if errors.Is(err, repository.ErrTripCompleted) {
			h.respondWithError(w, r, http.StatusConflict, "trip_completed", "Conflict", "Trip already completed")
			return
		}
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	trip, err := h.repo.GetTrip(r.Context(), tripStr)
	if err != nil {
		h.respondWithError(w, r, http.StatusInternalServerError, "internal_error", "Internal Error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}

// 4. GET /health - Системная ручка
func (h *TripHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"OK"}`))
}

// 5. GET /ready - Системная ручка
func (h *TripHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	h.respondWithError(w, r, http.StatusNotImplemented, "not_implemented", "Not Implemented", "Use main ready handler")
}

// Заглушки
func (h *TripHandler) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *TripHandler) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId uuid.UUID) {
	w.WriteHeader(http.StatusNotImplemented)
}
