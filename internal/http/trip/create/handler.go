// Package create implements the HTTP handler for creating a trip.
package create

import (
	"net/http"

	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
)

type Handler struct {
	usecase tripCreator
	metrics metrics
	logger  logger
}

func New(usecase tripCreator, metrics metrics, logger logger) *Handler {
	return &Handler{usecase: usecase, metrics: metrics, logger: logger}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	body, requestHash, err := decodeRequest(r)
	if err != nil {
		httputil.WriteProblem(w, r, http.StatusBadRequest, httputil.Problem("invalid_request", "Invalid request", err.Error(), http.StatusBadRequest))
		return
	}

	result, err := h.usecase.Execute(r.Context(), toInput(body, params.IdempotencyKey, requestHash))
	if err != nil {
		status, problem := domainerror.Problem(err)
		httputil.WriteProblem(w, r, status, problem)

		return
	}

	if result.Created {
		h.metrics.IncCreated(r.Context())
		h.logger.InfoContext(r.Context(), "trip created", "trip_id", result.Trip.ID)
		w.Header().Set("Location", "/api/v1/trips/"+result.Trip.ID.String())
		httputil.WriteJSON(w, http.StatusCreated, httputil.ToAPI(result.Trip))

		return
	}

	httputil.WriteJSON(w, http.StatusOK, httputil.ToAPI(result.Trip))
}
