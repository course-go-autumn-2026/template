package get

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/XrystikLK/template/internal/http/domainerror"
)

type Handler struct {
	uc useCase
}

func New(uc useCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, tripID uuid.UUID) {
	tripModel, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(toAPITrip(tripModel))
}
