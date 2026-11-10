package get

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/murmyauuu/template/api"
	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
	httpx "github.com/murmyauuu/template/internal/http"
	"github.com/murmyauuu/template/internal/http/domainerror"
)

type usecase interface {
	Execute(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error)
}

type Handler struct {
	Usecase usecase
	Logger  *slog.Logger
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	value, err := h.Usecase.Execute(r.Context(), tripID)
	if err != nil {
		status, body := domainerror.Problem(err)
		body.Instance = &r.URL.Path
		httpx.WriteProblem(w, status, body)
		if h.Logger != nil {
			h.Logger.ErrorContext(r.Context(), "trip operation failed", "error", err)
		}
		return
	}
	httpx.WriteJSON(w, http.StatusOK, "application/json", toAPI(value))
}
