package create

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/metric"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/http/domainerror"
)

type Handler struct {
	uc           usecase
	logger       *slog.Logger
	tripsCreated metric.Int64Counter
}

func New(uc usecase, logger *slog.Logger, tripsCreated metric.Int64Counter) *Handler {
	return &Handler{
		uc:           uc,
		logger:       logger,
		tripsCreated: tripsCreated,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		domainerror.Write(w, r, trip.NewValidationError("body", err.Error()))
		return
	}

	res, err := h.uc.Execute(r.Context(), toUsecaseRequest(req))
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}

	if h.tripsCreated != nil {
		h.tripsCreated.Add(r.Context(), 1)
	}
	h.logger.InfoContext(r.Context(), "trip created", slog.String("trip_id", res.ID.String()))

	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", res.ID))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(ToAPITrip(res))
}
