package create

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel/metric"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/http/domainerror"
)

type Handler struct {
	uc             usecase
	logger         *slog.Logger
	positionsSaved metric.Int64Counter
}

func New(uc usecase, logger *slog.Logger, positionsSaved metric.Int64Counter) *Handler {
	return &Handler{
		uc:             uc,
		logger:         logger,
		positionsSaved: positionsSaved,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	var req api.CreateTripPositionJSONRequestBody
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		domainerror.Write(w, r, domaintrip.NewValidationError("body", err.Error()))
		return
	}

	res, err := h.uc.Execute(r.Context(), toUsecaseRequest(tripID, req))
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}

	if h.positionsSaved != nil {
		h.positionsSaved.Add(r.Context(), 1)
	}
	h.logger.InfoContext(r.Context(), "trip position saved",
		slog.String("trip_id", res.TripID.String()),
		slog.Int64("position_id", res.ID),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAPIPosition(res))
}
