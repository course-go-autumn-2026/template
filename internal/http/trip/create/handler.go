package create

import (
	"log/slog"
	"net/http"

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
	data, err := decodeCreateTrip(r.Body)
	if err != nil {
		respond.Invalid(w, r, h.logger, err.Error())

		return
	}

	entity, err := h.usecase.Execute(r.Context(), convert.Data(data))
	if err != nil {
		respond.Error(w, r, h.logger, err)

		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+entity.ID.String())
	respond.JSON(w, r, h.logger, http.StatusCreated, "application/json", convert.Response(entity))
}
