package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/riiin11/labs/internal/generated"
	"github.com/riiin11/labs/internal/repository"
	"github.com/riiin11/labs/internal/txmanager"
)

type Handlers struct {
	pool *pgxpool.Pool
	repo *repository.TripRepo
	txm  txmanager.TxManager
}

func NewHandlers(pool *pgxpool.Pool, repo *repository.TripRepo, txm txmanager.TxManager) *Handlers {
	return &Handlers{pool: pool, repo: repo, txm: txm}
}

func (h *Handlers) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	var body api.TripData
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeProblem(w, r, 400, "invalid_request", "invalid json body")
		return
	}

	id := uuid.New()
	t := repository.Trip{
		ID:        id,
		UserID:    body.UserId,
		DriverID:  body.DriverId,
		StartLat:  body.StartPoint.Latitude,
		StartLon:  body.StartPoint.Longitude,
		EndLat:    body.EndPoint.Latitude,
		EndLon:    body.EndPoint.Longitude,
		Price:     body.Price,
		Status:    "active",
		StartedAt: time.Now().UTC(),
	}

	var created repository.Trip
	err := h.txm.Do(r.Context(), func(ctx context.Context) error {
		if err := h.repo.Insert(ctx, t); err != nil {
			return err
		}
		var err error
		created, err = h.repo.Get(ctx, id)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDriverBusy):
			writeProblem(w, r, 409, "driver_busy", "driver already has an active trip")
		default:
			writeProblem(w, r, 500, "internal_error", "internal error")
		}
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+created.ID.String())
	writeJSON(w, 201, toTripDTO(created))
}

func (h *Handlers) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	t, err := h.repo.Get(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, repository.ErrTripNotFound) {
			writeProblem(w, r, 404, "trip_not_found", "trip not found")
			return
		}
		writeProblem(w, r, 500, "internal_error", "internal error")
		return
	}
	writeJSON(w, 200, toTripDTO(t))
}

func (h *Handlers) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	var finished repository.Trip
	err := h.txm.Do(r.Context(), func(ctx context.Context) error {
		var err error
		finished, err = h.repo.Finish(ctx, tripId)
		return err
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTripNotFound):
			writeProblem(w, r, 404, "trip_not_found", "trip not found")
		case errors.Is(err, repository.ErrTripCompleted):
			writeProblem(w, r, 409, "trip_completed", "trip already completed")
		default:
			writeProblem(w, r, 500, "internal_error", "internal error")
		}
		return
	}
	writeJSON(w, 200, toTripDTO(finished))
}

func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		writeJSON(w, 503, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, 200, api.HealthResponse{Status: api.Ok})
}

func (h *Handlers) ListTripPositions(w http.ResponseWriter, r *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h *Handlers) CreateTripPosition(w http.ResponseWriter, r *http.Request, _ api.TripId) {
	w.WriteHeader(http.StatusNotImplemented)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	inst := r.URL.Path
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    code,
		Status:   int32(status),
		Detail:   &detail,
		Instance: &inst,
		Code:     code,
	})
}

func toTripDTO(t repository.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.StartLat, Longitude: t.StartLon},
		EndPoint:   api.Coordinates{Latitude: t.EndLat, Longitude: t.EndLon},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
