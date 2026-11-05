package create

import (
	"encoding/json"
	"fmt"
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

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req api.TripData
	if err := decoder.Decode(&req); err != nil {
		domainerror.WriteWithDetail(w, http.StatusBadRequest, "invalid_request", "Invalid request body", err.Error())
		return
	}

	input := toModelInput(req)
	tr, err := h.usecase.Execute(r.Context(), input)
	if err != nil {
		domainerror.Write(w, err)
		return
	}

	resp := toAPITrip(tr)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", tr.ID))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
