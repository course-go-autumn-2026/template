package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	api "github.com/poiuyttq/tripgo/internal/generated"
	"github.com/poiuyttq/tripgo/internal/model"
	"github.com/poiuyttq/tripgo/internal/service"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func ReadyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
			return
		}

		writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
	}
}

func CreateTripHandler(tripService *service.TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req api.TripData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "Request body is invalid", r.URL.Path)
			return
		}
		if err := validateTripData(req); err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error(), r.URL.Path)
			return
		}

		trip := model.Trip{
			ID:        uuid.NewString(),
			UserID:    req.UserId.String(),
			DriverID:  req.DriverId.String(),
			StartLat:  req.StartPoint.Latitude,
			StartLng:  req.StartPoint.Longitude,
			EndLat:    req.EndPoint.Latitude,
			EndLng:    req.EndPoint.Longitude,
			Price:     req.Price,
			Status:    "active",
			StartedAt: time.Now().UTC(),
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}

		created, err := tripService.CreateTrip(r.Context(), trip)
		if err != nil {
			handleServiceError(w, r, err)
			return
		}

		resp := tripModelToAPI(created)
		w.Header().Set("Location", "/api/v1/trips/"+created.ID)
		writeJSON(w, http.StatusCreated, resp)
	}
}

func GetTripHandler(tripService *service.TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "tripId")
		if _, err := uuid.Parse(tripID); err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "tripId must be a valid UUID", r.URL.Path)
			return
		}

		trip, err := tripService.GetTrip(r.Context(), tripID)
		if err != nil {
			handleServiceError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, tripModelToAPI(trip))
	}
}

func FinishTripHandler(tripService *service.TripService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tripID := chi.URLParam(r, "tripId")
		if _, err := uuid.Parse(tripID); err != nil {
			writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", "tripId must be a valid UUID", r.URL.Path)
			return
		}

		if err := tripService.FinishTrip(r.Context(), tripID); err != nil {
			handleServiceError(w, r, err)
			return
		}

		trip, err := tripService.GetTrip(r.Context(), tripID)
		if err != nil {
			handleServiceError(w, r, err)
			return
		}

		writeJSON(w, http.StatusOK, tripModelToAPI(trip))
	}
}

func validateTripData(req api.TripData) error {
	if req.UserId == uuid.Nil || req.DriverId == uuid.Nil {
		return fmt.Errorf("user_id and driver_id are required")
	}
	if req.StartPoint.Latitude < -90 || req.StartPoint.Latitude > 90 {
		return fmt.Errorf("start_point.latitude must be between -90 and 90")
	}
	if req.StartPoint.Longitude < -180 || req.StartPoint.Longitude > 180 {
		return fmt.Errorf("start_point.longitude must be between -180 and 180")
	}
	if req.EndPoint.Latitude < -90 || req.EndPoint.Latitude > 90 {
		return fmt.Errorf("end_point.latitude must be between -90 and 90")
	}
	if req.EndPoint.Longitude < -180 || req.EndPoint.Longitude > 180 {
		return fmt.Errorf("end_point.longitude must be between -180 and 180")
	}
	if req.Price < 0 {
		return fmt.Errorf("price must be greater than or equal to 0")
	}
	return nil
}

func tripModelToAPI(trip model.Trip) api.Trip {
	tripID, err := uuid.Parse(trip.ID)
	if err != nil {
		tripID = uuid.Nil
	}
	userID, err := uuid.Parse(trip.UserID)
	if err != nil {
		userID = uuid.Nil
	}
	driverID, err := uuid.Parse(trip.DriverID)
	if err != nil {
		driverID = uuid.Nil
	}

	return api.Trip{
		Id:             api.TripId(tripID),
		UserId:         api.TripId(userID),
		DriverId:       api.TripId(driverID),
		StartPoint:     api.Coordinates{Latitude: trip.StartLat, Longitude: trip.StartLng},
		EndPoint:       api.Coordinates{Latitude: trip.EndLat, Longitude: trip.EndLng},
		Price:          trip.Price,
		Status:         api.TripStatus(trip.Status),
		StartedAt:      trip.StartedAt,
		FinishedAt:     trip.FinishedAt,
		LastPositionAt: nil,
	}
}

func handleServiceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, service.ErrDriverHasActiveTrip):
		writeProblem(w, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip", r.URL.Path)
	case errors.Is(err, service.ErrTripAlreadyCompleted):
		writeProblem(w, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip", r.URL.Path)
	case errors.Is(err, service.ErrTripNotFound), strings.Contains(err.Error(), "trip not found"):
		writeProblem(w, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found", r.URL.Path)
	default:
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error", r.URL.Path)
	}
}

func writeProblem(w http.ResponseWriter, status int, code, title, detail, instance string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	problem := api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}
	_ = json.NewEncoder(w).Encode(problem)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
