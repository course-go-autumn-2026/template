package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/razrabbesezdav-art/template/internal/generated"
	"github.com/razrabbesezdav-art/template/internal/repository"
	"github.com/razrabbesezdav-art/template/internal/service"
)

type Trips struct {
	api.Unimplemented

	service *service.Trips
	db      *pgxpool.Pool
}

func NewTrips(svc *service.Trips, db *pgxpool.Pool) *Trips {
	return &Trips{
		service: svc,
		db:      db,
	}
}

func (h *Trips) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var request api.TripData
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeProblem(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"Invalid request body",
		)
		return
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeProblem(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"Request must contain one JSON object",
		)
		return
	}

	if !validCoordinates(request.StartPoint) ||
		!validCoordinates(request.EndPoint) ||
		request.Price < 0 ||
		request.UserId.String() == uuid.Nil.String() ||
		request.DriverId.String() == uuid.Nil.String() {
		writeProblem(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"Invalid trip data",
		)
		return
	}

	userID, err := uuid.Parse(request.UserId.String())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid user_id")
		return
	}

	driverID, err := uuid.Parse(request.DriverId.String())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid driver_id")
		return
	}

	trip, err := h.service.Create(r.Context(), repository.TripData{
		UserID:   userID,
		DriverID: driverID,
		StartLat: request.StartPoint.Latitude,
		StartLon: request.StartPoint.Longitude,
		EndLat:   request.EndPoint.Latitude,
		EndLon:   request.EndPoint.Longitude,
		Price:    request.Price,
	})
	if err != nil {
		if errors.Is(err, repository.ErrDriverBusy) {
			writeProblem(
				w,
				http.StatusConflict,
				"driver_busy",
				"Driver already has an active trip",
			)
			return
		}

		writeProblem(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"Could not create trip",
		)
		return
	}

	writeJSON(w, http.StatusCreated, toAPITrip(trip))
}

func (h *Trips) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	id, err := uuid.Parse(tripID.String())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_trip_id", "Invalid trip ID")
		return
	}

	trip, err := h.service.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			writeProblem(w, http.StatusNotFound, "trip_not_found", "Trip not found")
			return
		}

		writeProblem(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"Could not get trip",
		)
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *Trips) FinishTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	id, err := uuid.Parse(tripID.String())
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_trip_id", "Invalid trip ID")
		return
	}

	trip, err := h.service.Finish(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTripNotFound):
			writeProblem(w, http.StatusNotFound, "trip_not_found", "Trip not found")
		case errors.Is(err, repository.ErrTripCompleted):
			writeProblem(w, http.StatusConflict, "trip_completed", "Trip is already completed")
		default:
			writeProblem(
				w,
				http.StatusInternalServerError,
				"internal_error",
				"Could not finish trip",
			)
		}
		return
	}

	writeJSON(w, http.StatusOK, toAPITrip(trip))
}

func (h *Trips) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status: api.HealthResponseStatus("ok"),
	})
}

func (h *Trips) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeProblem(
			w,
			http.StatusServiceUnavailable,
			"database_unavailable",
			"Database is unavailable",
		)
		return
	}

	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status: api.HealthResponseStatus("ok"),
	})
}

func validCoordinates(c api.Coordinates) bool {
	return isFinite(c.Latitude) &&
		isFinite(c.Longitude) &&
		c.Latitude >= -90 &&
		c.Latitude <= 90 &&
		c.Longitude >= -180 &&
		c.Longitude <= 180
}

func isFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func toAPITrip(trip repository.Trip) api.Trip {
	result := api.Trip{
		Id:       trip.ID,
		UserId:   trip.UserID,
		DriverId: trip.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  trip.StartLat,
			Longitude: trip.StartLon,
		},
		EndPoint: api.Coordinates{
			Latitude:  trip.EndLat,
			Longitude: trip.EndLon,
		},
		Price:     trip.Price,
		Status:    api.TripStatus(trip.Status),
		StartedAt: trip.StartedAt,
	}
	result.FinishedAt = trip.FinishedAt
	return result
}

func writeProblem(w http.ResponseWriter, status int, code, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:   "about:blank",
		Title:  title,
		Status: int32(status),
		Code:   code,
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(value)
}
