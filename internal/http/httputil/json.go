// Package httputil holds the JSON response helpers shared by the trip handlers.
package httputil

import (
	"encoding/json"
	"net/http"

	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// WriteProblem writes an RFC 9457 problem response, filling Instance from the request path.
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, problem api.Problem) {
	instance := r.URL.Path
	problem.Instance = &instance

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem)
}

// Problem builds an RFC 9457 problem body for errors detected in the HTTP layer itself.
func Problem(code, title, detail string, status int) api.Problem {
	return api.Problem{
		Type:   "https://tripgo.example/problems/" + code,
		Title:  title,
		Status: int32(status), //nolint:gosec // status is always a fixed http.Status* constant passed by callers
		Detail: &detail,
		Code:   code,
	}
}

// ToAPI maps the cross-layer trip model to the generated API representation.
func ToAPI(t model.Trip) api.Trip {
	return api.Trip{
		Id:             t.ID,
		UserId:         t.UserID,
		DriverId:       t.DriverID,
		StartPoint:     api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:       api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
		Price:          t.Price,
		Status:         api.TripStatus(t.Status),
		StartedAt:      t.StartedAt,
		FinishedAt:     t.FinishedAt,
		LastPositionAt: t.LastPositionAt,
	}
}
