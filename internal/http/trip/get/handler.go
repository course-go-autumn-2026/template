package get

import (
	"encoding/json"
	"net/http"

	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
)

type Handler struct {
	usecase getUsecase
}

func New(usecase getUsecase) *Handler {
	return &Handler{usecase: usecase}
}

func (h *Handler) Handle(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	foundTrip, err := h.usecase.Execute(r.Context(), tripID)
	if err != nil {
		domainerror.Write(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toResponse(foundTrip))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
