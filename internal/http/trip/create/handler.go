package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
)

type Handler struct {
	usecase createUsecase
}

func New(usecase createUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(
	w http.ResponseWriter,
	r *http.Request,
) {
	tripData, err := decodeCreateTripRequest(r.Body)
	if err != nil {
		domainerror.Write(w, fmt.Errorf("decode create trip request: %w", trip.ErrInvalidRequest))
		return
	}

	createdTrip, err := h.usecase.Execute(r.Context(), toCreateTripData(tripData))
	if err != nil {
		domainerror.Write(w, err)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+createdTrip.ID.String())
	writeJSON(w, http.StatusCreated, toResponse(createdTrip))
}

func decodeCreateTripRequest(body io.Reader) (api.TripData, error) {
	var raw json.RawMessage
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&raw); err != nil {
		return api.TripData{}, fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.TripData{}, errors.New("request body must contain one JSON value")
	}

	fields, err := requiredJSONObject(raw, "user_id", "driver_id", "start_point", "end_point", "price")
	if err != nil {
		return api.TripData{}, err
	}
	if _, err = requiredJSONObject(fields["start_point"], "latitude", "longitude"); err != nil {
		return api.TripData{}, fmt.Errorf("validate start_point: %w", err)
	}
	if _, err = requiredJSONObject(fields["end_point"], "latitude", "longitude"); err != nil {
		return api.TripData{}, fmt.Errorf("validate end_point: %w", err)
	}

	var tripData api.TripData
	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err = strictDecoder.Decode(&tripData); err != nil {
		return api.TripData{}, fmt.Errorf("decode trip data: %w", err)
	}

	if tripData.UserId == uuid.Nil || tripData.DriverId == uuid.Nil ||
		!coordinatesValid(tripData.StartPoint) ||
		!coordinatesValid(tripData.EndPoint) ||
		tripData.Price < 0 {
		return api.TripData{}, errors.New("trip data failed validation")
	}

	return tripData, nil
}

func requiredJSONObject(raw json.RawMessage, requiredFields ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, errors.New("JSON object is required")
	}
	for _, field := range requiredFields {
		value, ok := fields[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("field %q is required", field)
		}
	}

	return fields, nil
}

func coordinatesValid(coordinates api.Coordinates) bool {
	return coordinates.Latitude >= -90 && coordinates.Latitude <= 90 &&
		coordinates.Longitude >= -180 && coordinates.Longitude <= 180
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
