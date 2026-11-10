package create

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/google/uuid"

	"github.com/murmyauuu/template/api"
	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	httpx "github.com/murmyauuu/template/internal/http"
	"github.com/murmyauuu/template/internal/http/domainerror"
)

const maxTripBodySize = 64 * 1024

type Handler struct {
	Usecase usecase
	Logger  *slog.Logger
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	data, err := decodeTripData(r.Context(), w, r)
	if err != nil {
		status, body := domainerror.Problem(domaintrip.ValidationError{Reason: "Request body is invalid"})
		body.Instance = &r.URL.Path
		httpx.WriteProblem(w, status, body)
		return
	}
	out, err := h.Usecase.Execute(r.Context(), toInput(data), params.IdempotencyKey)
	if err != nil {
		status, body := domainerror.Problem(err)
		body.Instance = &r.URL.Path
		httpx.WriteProblem(w, status, body)
		if h.Logger != nil {
			h.Logger.ErrorContext(r.Context(), "trip operation failed", "error", err)
		}
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+out.Trip.Id.String())
	status := http.StatusCreated
	if !out.Fresh {
		status = http.StatusOK
	}
	httpx.WriteJSON(w, status, "application/json", toAPI(out.Trip))
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
