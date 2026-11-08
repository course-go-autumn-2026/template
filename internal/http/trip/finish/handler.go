package finish

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/whxtelyy/template/internal/http/trip/shared"
)

type Handler struct {
	usecase usecase
	logger  *slog.Logger
}

func New(uc usecase, logger *slog.Logger) *Handler { return &Handler{usecase: uc, logger: logger} }
func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	trip, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		shared.WriteServiceError(w, r, err, h.logger)
		return
	}
	shared.WriteJSON(w, http.StatusOK, shared.ToAPITrip(trip))
	h.logger.InfoContext(r.Context(), "trip finished", "trip_id", trip.ID.String())
}
