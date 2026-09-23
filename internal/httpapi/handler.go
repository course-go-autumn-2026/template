package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/dumbcirno/diploma-mill-x-avito-golang-course/internal/generated"
	"github.com/dumbcirno/diploma-mill-x-avito-golang-course/internal/trip"
)

type Handler struct {
	svc          *trip.Service
	pool         *pgxpool.Pool
	queryTimeout time.Duration
}

func NewHandler(svc *trip.Service, pool *pgxpool.Pool, queryTimeout time.Duration) *Handler {
	return &Handler{svc: svc, pool: pool, queryTimeout: queryTimeout}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Failed to read request body")
		return
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var body api.TripData
	if err := dec.Decode(&body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
		return
	}
	if err := validateTripData(body); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	in := trip.CreateInput{
		UserID:   uuid.UUID(body.UserId),
		DriverID: uuid.UUID(body.DriverId),
		StartLat: body.StartPoint.Latitude,
		StartLon: body.StartPoint.Longitude,
		EndLat:   body.EndPoint.Latitude,
		EndLon:   body.EndPoint.Longitude,
		Price:    body.Price,
	}

	var key *uuid.UUID
	if params.IdempotencyKey != nil {
		id := uuid.UUID(*params.IdempotencyKey)
		if id == uuid.Nil {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Idempotency-Key must be a non-empty UUID")
			return
		}
		key = &id
	}

	sum := sha256.Sum256(raw)
	created, replay, err := h.svc.Create(r.Context(), in, key, hex.EncodeToString(sum[:]))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	status := http.StatusCreated
	if replay {
		status = http.StatusOK
	}
	if status == http.StatusCreated {
		w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	}
	writeJSON(w, status, toAPITrip(created))
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	id := uuid.UUID(tripId)
	if id == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "tripId must be a valid UUID")
		return
	}
	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITrip(t))
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	id := uuid.UUID(tripId)
	if id == uuid.Nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "tripId must be a valid UUID")
		return
	}
	t, err := h.svc.Finish(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAPITrip(t))
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func validateTripData(body api.TripData) error {
	if uuid.UUID(body.UserId) == uuid.Nil {
		return errors.New("user_id must be a non-empty UUID")
	}
	if uuid.UUID(body.DriverId) == uuid.Nil {
		return errors.New("driver_id must be a non-empty UUID")
	}
	if body.StartPoint.Latitude < -90 || body.StartPoint.Latitude > 90 ||
		body.EndPoint.Latitude < -90 || body.EndPoint.Latitude > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if body.StartPoint.Longitude < -180 || body.StartPoint.Longitude > 180 ||
		body.EndPoint.Longitude < -180 || body.EndPoint.Longitude > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	if body.Price < 0 {
		return errors.New("price must be greater than or equal to 0")
	}
	return nil
}

func toAPITrip(t trip.Trip) api.Trip {
	started := t.StartedAt.UTC()
	var finished *time.Time
	if t.FinishedAt != nil {
		utc := t.FinishedAt.UTC()
		finished = &utc
	}
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartLat, Longitude: t.StartLon},
		EndPoint:   api.Coordinates{Latitude: t.EndLat, Longitude: t.EndLon},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  started,
		FinishedAt: finished,
	}
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, trip.ErrInvalidRequest):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
	case errors.Is(err, trip.ErrTripNotFound):
		writeProblem(w, r, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
	case errors.Is(err, trip.ErrTripCompleted):
		writeProblem(w, r, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
	case errors.Is(err, trip.ErrDriverBusy):
		writeProblem(w, r, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, trip.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusConflict, "idempotency_conflict", "Idempotency conflict", "Idempotency-Key was already used with a different request body")
	default:
		log.Printf("internal error: %v", err)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error")
	}
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, title, detail string) {
	instance := r.URL.Path
	body := api.Problem{
		Type:     "https://tripgo.example/problems/" + kebab(code),
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("write problem: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func kebab(code string) string {
	switch code {
	case "invalid_request":
		return "invalid-request"
	case "trip_not_found":
		return "trip-not-found"
	case "trip_completed":
		return "trip-completed"
	case "driver_busy":
		return "driver-busy"
	case "idempotency_conflict":
		return "idempotency-conflict"
	case "internal_error":
		return "internal-error"
	default:
		return code
	}
}

func InvalidParam(w http.ResponseWriter, r *http.Request, err error) {
	writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request validation failed")
	_ = err
}
