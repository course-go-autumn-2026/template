package create

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	api "github.com/whxtelyy/template/internal/generated"
)

func decodeTripData(r *http.Request) (api.TripData, error) {
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&raw); err != nil {
		return api.TripData{}, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return api.TripData{}, err
	}
	if raw == nil {
		return api.TripData{}, errors.New("request body must be a JSON object")
	}

	allowed := map[string]struct{}{
		"user_id": {}, "driver_id": {}, "start_point": {}, "end_point": {}, "price": {},
	}
	for field := range raw {
		if _, ok := allowed[field]; !ok {
			return api.TripData{}, fmt.Errorf("unknown field %q", field)
		}
	}
	for _, field := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		if _, ok := raw[field]; !ok {
			return api.TripData{}, fmt.Errorf("field %q is required", field)
		}
	}

	if err := validateCoordinateJSON("start_point", raw["start_point"]); err != nil {
		return api.TripData{}, err
	}
	if err := validateCoordinateJSON("end_point", raw["end_point"]); err != nil {
		return api.TripData{}, err
	}
	if string(raw["price"]) == "null" {
		return api.TripData{}, errors.New("field \"price\" cannot be null")
	}

	var data api.TripData
	body, err := json.Marshal(raw)
	if err != nil {
		return api.TripData{}, fmt.Errorf("marshal request: %w", err)
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return api.TripData{}, fmt.Errorf("decode request: %w", err)
	}
	if data.UserId == uuid.Nil || data.DriverId == uuid.Nil {
		return api.TripData{}, errors.New("user_id and driver_id must be non-empty UUIDs")
	}
	if err := validateCoordinates("start_point", data.StartPoint); err != nil {
		return api.TripData{}, err
	}
	if err := validateCoordinates("end_point", data.EndPoint); err != nil {
		return api.TripData{}, err
	}
	if data.Price < 0 {
		return api.TripData{}, errors.New("price must be greater than or equal to zero")
	}
	return data, nil
}

func validateCoordinateJSON(name string, value json.RawMessage) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(value, &raw); err != nil || raw == nil {
		return fmt.Errorf("%s must be a JSON object", name)
	}
	allowed := map[string]struct{}{"latitude": {}, "longitude": {}}
	for field := range raw {
		if _, ok := allowed[field]; !ok {
			return fmt.Errorf("unknown field %q in %s", field, name)
		}
	}
	for _, field := range []string{"latitude", "longitude"} {
		value, ok := raw[field]
		if !ok {
			return fmt.Errorf("field %q in %s is required", field, name)
		}
		if string(value) == "null" {
			return fmt.Errorf("field %q in %s cannot be null", field, name)
		}
		var number float64
		if err := json.Unmarshal(value, &number); err != nil {
			return fmt.Errorf("field %q in %s must be a number", field, name)
		}
	}
	return nil
}

func validateCoordinates(name string, point api.Coordinates) error {
	if point.Latitude < -90 || point.Latitude > 90 {
		return fmt.Errorf("%s.latitude must be between -90 and 90", name)
	}
	if point.Longitude < -180 || point.Longitude > 180 {
		return fmt.Errorf("%s.longitude must be between -180 and 180", name)
	}
	return nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return fmt.Errorf("invalid trailing JSON: %w", err)
	}
	return nil
}

func hashTripData(data api.TripData) (string, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal trip data for idempotency: %w", err)
	}

	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
