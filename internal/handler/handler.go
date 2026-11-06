package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"

	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/repository"
	"github.com/xotapy-main/courseGo/internal/txmanager"
)

var _ api.ServerInterface = (*Handler)(nil)

type Handler struct {
	tripRepo    *repository.TripRepository
	historyRepo *repository.HistoryRepository
	txManager   txmanager.TxManager
	pool        *pgxpool.Pool
}

func New(
	tripRepo *repository.TripRepository,
	historyRepo *repository.HistoryRepository,
	txManager txmanager.TxManager,
	pool *pgxpool.Pool,
) *Handler {
	return &Handler{
		tripRepo:    tripRepo,
		historyRepo: historyRepo,
		txManager:   txManager,
		pool:        pool,
	}
}

func (h *Handler) writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	status32 := int32(status)
	instance := r.URL.Path

	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     fmt.Sprintf("https://tripgo.example/problems/%s", code),
		Title:    title,
		Status:   status32,
		Detail:   &detail,
		Instance: &instance,
		Code:     code,
	})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.HealthResponseStatus("ok"),
	})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(api.HealthResponse{
			Status: api.HealthResponseStatus("unavailable"),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.HealthResponseStatus("ok"),
	})
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var req api.TripData
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		h.writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	userUUID := uuid.UUID(req.UserId)
	driverUUID := uuid.UUID(req.DriverId)

	if userUUID == uuid.Nil || driverUUID == uuid.Nil || req.Price < 0 {
		h.writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Validation failed")
		return
	}

	now := time.Now().UTC()
	tripID := uuid.New()

	trip := &model.Trip{
		ID:         tripID,
		UserID:     userUUID,
		DriverID:   driverUUID,
		StartPoint: model.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		EndPoint:   model.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:      req.Price,
		Status:     "active",
		StartedAt:  now,
	}

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := h.tripRepo.Create(ctx, trip); err != nil {
			return err
		}

		history := &model.TripStatusHistory{
			TripID:    trip.ID,
			Status:    trip.Status,
			ChangedAt: now,
		}
		return h.historyRepo.Insert(ctx, history)
	})

	if err != nil {
		if errors.Is(err, model.ErrDriverBusy) {
			h.writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
			return
		}
		h.writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", trip.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toOpenAPITrip(trip))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	tripUUID := uuid.UUID(tripId)
	trip, err := h.tripRepo.GetByID(r.Context(), tripUUID)
	if err != nil {
		if errors.Is(err, model.ErrTripNotFound) {
			h.writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip not found")
			return
		}
		h.writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toOpenAPITrip(trip))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	now := time.Now().UTC()
	tripUUID := uuid.UUID(tripId)
	var finishedTrip *model.Trip

	err := h.txManager.Do(r.Context(), func(ctx context.Context) error {
		var err error
		finishedTrip, err = h.tripRepo.Finish(ctx, tripUUID, now)
		if err != nil {
			return err
		}

		history := &model.TripStatusHistory{
			TripID:    tripUUID,
			Status:    "completed",
			ChangedAt: now,
		}
		return h.historyRepo.Insert(ctx, history)
	})

	if err != nil {
		switch {
		case errors.Is(err, model.ErrTripNotFound):
			h.writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip not found")
		case errors.Is(err, model.ErrTripCompleted):
			h.writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Trip is already completed")
		default:
			h.writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal error", "Internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toOpenAPITrip(finishedTrip))
}

func toOpenAPITrip(t *model.Trip) api.Trip {
	resp := api.Trip{
		Id:       openapi_types.UUID(t.ID),
		UserId:   openapi_types.UUID(t.UserID),
		DriverId: openapi_types.UUID(t.DriverID),
		StartPoint: api.Coordinates{
			Latitude:  t.StartPoint.Latitude,
			Longitude: t.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  t.EndPoint.Latitude,
			Longitude: t.EndPoint.Longitude,
		},
		Price:     t.Price,
		Status:    api.TripStatus(t.Status),
		StartedAt: t.StartedAt,
	}
	if t.FinishedAt != nil {
		resp.FinishedAt = t.FinishedAt
	}
	if t.LastPositionAt != nil {
		resp.LastPositionAt = t.LastPositionAt
	}
	return resp
}
