package finish

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/otel/metric"

	"github.com/xotapy-main/courseGo/internal/http/domainerror"
)

type Handler struct {
	uc             usecase
	logger         *slog.Logger
	tripsCompleted metric.Int64Counter
}

func New(uc usecase, logger *slog.Logger, tripsCompleted metric.Int64Counter) *Handler {
	return &Handler{
		uc:             uc,
		logger:         logger,
		tripsCompleted: tripsCompleted,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID openapi_types.UUID) {
	res, err := h.uc.Execute(r.Context(), uuid.UUID(tripID))
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}

	if h.tripsCompleted != nil {
		h.tripsCompleted.Add(r.Context(), 1)
	}
	h.logger.InfoContext(r.Context(), "trip completed", slog.String("trip_id", res.ID.String()))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toAPITrip(res))
}
