package get

import (
	"encoding/json"
	"net/http"

	api "github.com/eternal-despair/template/internal/generated"
	"github.com/eternal-despair/template/internal/http/domainerror"
)

type Handler struct {
	usecase usecase
}

func New(uc usecase) *Handler {
	return &Handler{usecase: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	tr, err := h.usecase.Execute(r.Context(), tripId)
	if err != nil {
		domainerror.Write(w, err)
		return
	}

	resp := toAPITrip(tr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
