package finish

import (
	"context"
	"errors"
	"io"
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

func (h *Handler) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	var firstByte [1]byte
	n, err := io.ReadFull(r.Body, firstByte[:])
	if n != 0 || (err != nil && !errors.Is(err, io.EOF)) {
		status, body := domainerror.Problem(domaintrip.ValidationError{Reason: "Request body must be empty"})
		body.Instance = &r.URL.Path
		httpx.WriteProblem(w, status, body)
		return
	}
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
