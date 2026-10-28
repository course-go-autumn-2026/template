package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	api "github.com/Yolanesso/avito-labs/internal/generated"
	"github.com/Yolanesso/avito-labs/internal/service"
)

type Handler struct {
	api.Unimplemented

	pingDatabase func(context.Context) error
	queryTimeout time.Duration
	tripService  *service.TripService
}

func NewHandler(
	pingDatabase func(context.Context) error,
	queryTimeout time.Duration,
	tripService *service.TripService,
) *Handler {
	return &Handler{
		pingDatabase: pingDatabase,
		queryTimeout: queryTimeout,
		tripService:  tripService,
	}
}

func (h *Handler) Health(
	w http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status: api.Ok,
	})
}

func (h *Handler) Ready(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(
		r.Context(),
		h.queryTimeout,
	)
	defer cancel()

	err := h.pingDatabase(ctx)
	if err != nil {
		writeJSON(
			w,
			http.StatusServiceUnavailable,
			api.HealthResponse{
				Status: api.Unavailable,
			},
		)
		return
	}

	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status: api.Ok,
	})
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	value any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(value)
	if err != nil {
		return
	}
}
