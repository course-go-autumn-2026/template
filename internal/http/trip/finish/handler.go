package finish

import (
	"net/http"
	"time"

	api "github.com/toooriii/tori/internal/generated"
	"github.com/toooriii/tori/internal/http/domainerror"
)

type Handler struct {
	uc UseCase
}

func New(uc UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	t, err := h.uc.Do(r.Context(), tripID, time.Now().UTC())
	if err != nil {
		status, p := domainerror.Problem(err)
		writeProblemBody(w, status, p)
		return
	}
	writeJSON(w, http.StatusOK, toAPI(t))
}
