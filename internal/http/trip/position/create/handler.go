package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
)

type Handler struct {
	useCase usecase
}

func New(useCase usecase) *Handler {
	return &Handler{useCase: useCase}
}

func (h *Handler) Handle(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	positionData, err := decodePositionData(r.Body)
	if err != nil {
		domainerror.Write(w, trip.ErrInvalidRequest)
		return
	}

	position, err := h.useCase.Execute(r.Context(), toCreatePosition(tripID, positionData))
	if err != nil {
		domainerror.Write(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, toResponse(position))
}

func decodePositionData(body io.Reader) (api.PositionData, error) {
	var rowJson json.RawMessage
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&rowJson); err != nil {
		return api.PositionData{}, fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.PositionData{}, fmt.Errorf("request body must contain one JSON value")
	}

	fields, err := requiredJSONObject(rowJson, "coordinates", "recorded_at")
	if err != nil {
		return api.PositionData{}, fmt.Errorf("read json object: %w", err)
	}
	if _, err := requiredJSONObject(fields["coordinates"], "latitude", "longitude"); err != nil {
		return api.PositionData{}, fmt.Errorf("validate coordinates: %w", err)
	}

	var positionData api.PositionData
	strictDecoder := json.NewDecoder(bytes.NewReader(rowJson))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(&positionData); err != nil {
		return api.PositionData{}, fmt.Errorf("decode json object: %w", err)
	}

	if !coordinatesValid(positionData.Coordinates) || positionData.RecordedAt.IsZero() ||
		positionData.RecordedAt.After(time.Now().Add(time.Minute)) {
		return api.PositionData{}, fmt.Errorf("invalid position data: coordinates=%+v recorded_at=%v", positionData.Coordinates, positionData.RecordedAt)
	}
	return positionData, nil
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

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
