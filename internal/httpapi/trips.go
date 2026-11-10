package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/trip"
)

const maxTripBodySize = 64 * 1024

func (h *handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	data, err := decodeTripData(r.Context(), w, r)
	if err != nil {
		h.problem(r.Context(), w, r.URL.Path, http.StatusBadRequest, "invalid_request", "Invalid request", "Request body is invalid")
		return
	}
	created, fresh, err := h.trips.Create(r.Context(), data, params.IdempotencyKey)
	if err != nil {
		h.tripError(r.Context(), w, r.URL.Path, err)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+created.Id.String())
	status := http.StatusCreated
	if !fresh {
		status = http.StatusOK
	}
	h.writeJSON(r.Context(), w, status, "application/json", created)
}

func (h *handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	value, err := h.trips.Get(r.Context(), tripID)
	if err != nil {
		h.tripError(r.Context(), w, r.URL.Path, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, "application/json", value)
}

func (h *handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	// У операции завершения нет тела запроса.
	var firstByte [1]byte
	n, err := io.ReadFull(r.Body, firstByte[:])
	if n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
		h.problem(r.Context(), w, r.URL.Path, http.StatusBadRequest, "invalid_request", "Invalid request", "Request body must be empty")
		return
	}
	value, err := h.trips.Finish(r.Context(), tripID)
	if err != nil {
		h.tripError(r.Context(), w, r.URL.Path, err)
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, "application/json", value)
}

func decodeTripData(ctx context.Context, w http.ResponseWriter, r *http.Request) (api.TripData, error) {
	if err := ctx.Err(); err != nil {
		return api.TripData{}, err
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return api.TripData{}, fmt.Errorf("expected application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxTripBodySize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return api.TripData{}, fmt.Errorf("read request body: %w", err)
	}
	fields, ok := requiredObject(body, "user_id", "driver_id", "start_point", "end_point", "price")
	if !ok {
		return api.TripData{}, fmt.Errorf("expected required trip fields only")
	}
	for _, name := range []string{"start_point", "end_point"} {
		if _, ok := requiredObject(fields[name], "latitude", "longitude"); !ok {
			return api.TripData{}, fmt.Errorf("expected both coordinate fields only")
		}
	}
	var data api.CreateTripJSONRequestBody
	if err := json.Unmarshal(body, &data); err != nil {
		return api.TripData{}, fmt.Errorf("decode trip: %w", err)
	}
	if data.UserId == uuid.Nil || data.DriverId == uuid.Nil || data.Price < 0 ||
		!validCoordinates(data.StartPoint) || !validCoordinates(data.EndPoint) {
		return api.TripData{}, fmt.Errorf("trip fields are outside allowed values")
	}
	return data, nil
}

// Сгенерированные типы содержат значения, поэтому сами по себе не различают
// отсутствующее поле, null и допустимый ноль. Проверяем присутствие до декодирования.
func requiredObject(raw json.RawMessage, names ...string) (map[string]json.RawMessage, bool) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) != len(names) {
		return nil, false
	}
	for _, name := range names {
		value, exists := fields[name]
		if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, false
		}
	}
	return fields, true
}

func validCoordinates(value api.Coordinates) bool {
	return value.Latitude >= -90 && value.Latitude <= 90 && value.Longitude >= -180 && value.Longitude <= 180
}

func (h *handler) tripError(ctx context.Context, w http.ResponseWriter, instance string, err error) {
	switch {
	case errors.Is(err, trip.ErrDriverBusy):
		h.problem(ctx, w, instance, http.StatusConflict, "driver_busy", "Driver busy", "Driver already has an active trip")
	case errors.Is(err, trip.ErrTripNotFound):
		h.problem(ctx, w, instance, http.StatusNotFound, "trip_not_found", "Trip not found", "Trip was not found")
	case errors.Is(err, trip.ErrTripCompleted):
		h.problem(ctx, w, instance, http.StatusConflict, "trip_completed", "Trip completed", "Operation is not allowed for a completed trip")
	case errors.Is(err, trip.ErrIdempotencyConflict):
		h.problem(ctx, w, instance, http.StatusConflict, "idempotency_conflict", "Idempotency conflict", "Idempotency key was already used with different request data")
	default:
		h.logger.ErrorContext(ctx, "trip operation failed", "path", instance, "error", err)
		h.problem(ctx, w, instance, http.StatusInternalServerError, "internal_error", "Internal server error", "Unable to process the request")
	}
}
