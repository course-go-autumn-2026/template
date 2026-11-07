package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/XrystikLK/template/internal/db"
	api "github.com/XrystikLK/template/internal/generated"
	"github.com/XrystikLK/template/internal/repository"
	"github.com/google/uuid"
)

type TripRepository interface {
	CreateTrip(ctx context.Context, trip *repository.Trip) error
	GetTripByID(ctx context.Context, id uuid.UUID) (*repository.Trip, error)
	FinishTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*repository.Trip, error)
	AddStatusHistory(ctx context.Context, history *repository.StatusHistory) error
}

type Pinger interface {
	Ping(ctx context.Context) error
}

type TripHandler struct {
	api.Unimplemented
	repo        TripRepository
	txManager   db.TxManager
	pinger      Pinger
	pingTimeout time.Duration
}

func NewTripHandler(repo TripRepository, txManager db.TxManager, pinger Pinger) *TripHandler {
	return &TripHandler{
		repo:        repo,
		txManager:   txManager,
		pinger:      pinger,
		pingTimeout: 2 * time.Second,
	}
}

func WriteProblem(w http.ResponseWriter, status int, code, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	p := api.Problem{
		Code:   code,
		Status: int32(status),
		Title:  title,
		Type:   fmt.Sprintf("https://tripgo.example/problems/%s", code),
	}
	if detail != "" {
		p.Detail = &detail
	}
	if instance != "" {
		p.Instance = &instance
	}

	_ = json.NewEncoder(w).Encode(p)
}

func toAPITrip(t *repository.Trip) api.Trip {
	return api.Trip{
		Id:       t.ID,
		UserId:   t.UserID,
		DriverId: t.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  t.StartLatitude,
			Longitude: t.StartLongitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndLatitude,
			Longitude: t.EndLongitude,
		},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: nil,
	}
}

type rawCoordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type rawCreateTripRequest struct {
	UserID     *uuid.UUID      `json:"user_id"`
	DriverID   *uuid.UUID      `json:"driver_id"`
	StartPoint *rawCoordinates `json:"start_point"`
	EndPoint   *rawCoordinates `json:"end_point"`
	Price      *int64          `json:"price"`
}

func (h *TripHandler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req rawCreateTripRequest
	if err := decoder.Decode(&req); err != nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error(), r.URL.Path)
		return
	}

	// Проверяем, что после первого JSON-объекта нет мусора или второго объекта
	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "unexpected data after JSON object", r.URL.Path)
		return
	}

	// Проверка наличия обязательных полей
	if req.UserID == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "user_id is required", r.URL.Path)
		return
	}
	if *req.UserID == uuid.Nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "user_id cannot be empty", r.URL.Path)
		return
	}

	if req.DriverID == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "driver_id is required", r.URL.Path)
		return
	}
	if *req.DriverID == uuid.Nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "driver_id cannot be empty", r.URL.Path)
		return
	}

	if req.StartPoint == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "start_point is required", r.URL.Path)
		return
	}
	if req.StartPoint.Latitude == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "start_point.latitude is required", r.URL.Path)
		return
	}
	if *req.StartPoint.Latitude < -90 || *req.StartPoint.Latitude > 90 {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "start_point.latitude must be between -90 and 90", r.URL.Path)
		return
	}
	if req.StartPoint.Longitude == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "start_point.longitude is required", r.URL.Path)
		return
	}
	if *req.StartPoint.Longitude < -180 || *req.StartPoint.Longitude > 180 {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "start_point.longitude must be between -180 and 180", r.URL.Path)
		return
	}

	if req.EndPoint == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "end_point is required", r.URL.Path)
		return
	}
	if req.EndPoint.Latitude == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "end_point.latitude is required", r.URL.Path)
		return
	}
	if *req.EndPoint.Latitude < -90 || *req.EndPoint.Latitude > 90 {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "end_point.latitude must be between -90 and 90", r.URL.Path)
		return
	}
	if req.EndPoint.Longitude == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "end_point.longitude is required", r.URL.Path)
		return
	}
	if *req.EndPoint.Longitude < -180 || *req.EndPoint.Longitude > 180 {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "end_point.longitude must be between -180 and 180", r.URL.Path)
		return
	}

	if req.Price == nil {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "price is required", r.URL.Path)
		return
	}
	if *req.Price < 0 {
		WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "price must be greater than or equal to 0", r.URL.Path)
		return
	}

	now := time.Now().UTC()
	tripID := uuid.New()

	trip := &repository.Trip{
		ID:             tripID,
		UserID:         *req.UserID,
		DriverID:       *req.DriverID,
		StartLatitude:  *req.StartPoint.Latitude,
		StartLongitude: *req.StartPoint.Longitude,
		EndLatitude:    *req.EndPoint.Latitude,
		EndLongitude:   *req.EndPoint.Longitude,
		Price:          *req.Price,
		Status:         "active",
		StartedAt:      now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := h.repo.CreateTrip(ctx, trip); err != nil {
			return err
		}

		history := &repository.StatusHistory{
			TripID:     tripID,
			FromStatus: nil,
			ToStatus:   "active",
			ChangedAt:  now,
		}
		return h.repo.AddStatusHistory(ctx, history)
	})

	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			WriteProblem(w, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip", r.URL.Path)
			return
		}
		slog.ErrorContext(r.Context(), "failed to create trip", "error", err)
		WriteProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", trip.ID.String()))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAPITrip(trip))
}

func (h *TripHandler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := h.repo.GetTripByID(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			WriteProblem(w, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found", r.URL.Path)
			return
		}
		slog.ErrorContext(r.Context(), "failed to get trip", "error", err, "trip_id", tripId)
		WriteProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toAPITrip(trip))
}

func (h *TripHandler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	now := time.Now().UTC()
	var finishedTrip *repository.Trip

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		trip, err := h.repo.FinishTrip(ctx, tripId, now)
		if err != nil {
			return err
		}
		finishedTrip = trip

		fromStatus := "active"
		history := &repository.StatusHistory{
			TripID:     tripId,
			FromStatus: &fromStatus,
			ToStatus:   "completed",
			ChangedAt:  now,
		}
		return h.repo.AddStatusHistory(ctx, history)
	})

	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			WriteProblem(w, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found", r.URL.Path)
			return
		}
		if errors.Is(err, repository.ErrTripAlreadyCompleted) {
			WriteProblem(w, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip", r.URL.Path)
			return
		}
		slog.ErrorContext(r.Context(), "failed to finish trip", "error", err, "trip_id", tripId)
		WriteProblem(w, http.StatusInternalServerError, "internal_error", "Internal server error", "An unexpected error occurred", r.URL.Path)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toAPITrip(finishedTrip))
}

func (h *TripHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

func (h *TripHandler) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pingCtx, cancel := context.WithTimeout(r.Context(), h.pingTimeout)
	defer cancel()

	if err := h.pinger.Ping(pingCtx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Unavailable})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}
