package get

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/freyliee/template/internal/http/respond"
	"github.com/freyliee/template/internal/http/trip/convert"
)

type Handler struct {
	usecase usecase
	logger  *slog.Logger
}

func New(usecase usecase, logger *slog.Logger) *Handler {
	return &Handler{usecase: usecase, logger: logger}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		respond.Invalid(w, r, h.logger, "tripId must be a UUID")

		return
	}

	entity, err := h.usecase.Execute(r.Context(), id)
	if err != nil {
		respond.Error(w, r, h.logger, err)

		return
	}

	respond.JSON(w, r, h.logger, http.StatusOK, "application/json", convert.Response(entity))
}
