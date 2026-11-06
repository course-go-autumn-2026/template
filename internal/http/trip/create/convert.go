package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	api "github.com/toooriii/tori/internal/generated"
	"github.com/toooriii/tori/internal/model"
)

func decodeTripData(body []byte) (api.TripData, error) {
	var req api.TripData

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, fmt.Errorf("invalid JSON body: %w", err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, errors.New("invalid JSON body: unexpected data after object")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return req, fmt.Errorf("invalid JSON body: %w", err)
	}
	for _, k := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		if !present(raw[k]) {
			return req, fmt.Errorf("field %q is required", k)
		}
	}
	for _, k := range []string{"start_point", "end_point"} {
		var pt map[string]json.RawMessage
		if err := json.Unmarshal(raw[k], &pt); err != nil {
			return req, fmt.Errorf("field %q must be an object", k)
		}
		for _, c := range []string{"latitude", "longitude"} {
			if !present(pt[c]) {
				return req, fmt.Errorf("field %q.%s is required", k, c)
			}
		}
	}
	return req, nil
}

func present(v json.RawMessage) bool {
	t := bytes.TrimSpace(v)
	return len(t) > 0 && string(t) != "null"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	instance := r.URL.Path
	d := detail
	p := api.Problem{
		Type:     "https://tripgo.example/problems/" + code,
		Title:    titleOf(code),
		Status:   int32(status), //nolint:gosec
		Detail:   &d,
		Instance: &instance,
		Code:     code,
	}
	writeProblemBody(w, status, p)
}

func writeProblemBody(w http.ResponseWriter, status int, p api.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(p)
}

func titleOf(code string) string {
	switch code {
	case "invalid_request":
		return "Invalid request"
	case "trip_not_found":
		return "Trip not found"
	case "trip_completed":
		return "Trip completed"
	case "driver_busy":
		return "Driver busy"
	case "idempotency_conflict":
		return "Idempotency conflict"
	default:
		return "Internal Server Error"
	}
}

func toAPI(t model.Trip) api.Trip {
	return api.Trip{
		Id:         t.ID,
		UserId:     t.UserID,
		DriverId:   t.DriverID,
		StartPoint: api.Coordinates{Latitude: t.Start.Latitude, Longitude: t.Start.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.End.Latitude, Longitude: t.End.Longitude},
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
	}
}
