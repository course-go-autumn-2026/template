package create

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	api "github.com/whxtelyy/template/internal/generated"
	"github.com/whxtelyy/template/internal/http/trip/shared"
	createuc "github.com/whxtelyy/template/internal/usecase/trip/create"
)

type Handler struct {
	usecase usecase
	logger  *slog.Logger
}

func New(uc usecase, logger *slog.Logger) *Handler { return &Handler{usecase: uc, logger: logger} }

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	request, err := decodeTripData(r)
	if err != nil {
		shared.WriteProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error(), "https://tripgo.example/problems/invalid-request")
		return
	}
	var idempotencyKey *uuid.UUID
	var requestHash string
	if params.IdempotencyKey != nil {
		key := *params.IdempotencyKey
		idempotencyKey = &key
		requestHash, err = hashTripData(request)
		if err != nil {
			h.logger.ErrorContext(r.Context(), "failed to hash request", "error", err)
			shared.WriteProblem(w, r, http.StatusInternalServerError, "internal_error", "Internal Server Error", "Internal server error", "https://tripgo.example/problems/internal-error")
			return
		}
	}
	result, err := h.usecase.Execute(r.Context(), createuc.Request{
		UserID: request.UserId, DriverID: request.DriverId,
		StartPoint: domain.Coordinates{Latitude: request.StartPoint.Latitude, Longitude: request.StartPoint.Longitude},
		EndPoint:   domain.Coordinates{Latitude: request.EndPoint.Latitude, Longitude: request.EndPoint.Longitude},
		Price:      request.Price, IdempotencyKey: idempotencyKey, RequestHash: requestHash,
	})
	if err != nil {
		shared.WriteServiceError(w, r, err, h.logger)
		return
	}
	w.Header().Set("Location", "/api/v1/trips/"+result.Trip.ID.String())
	status := http.StatusCreated
	if result.Replayed {
		status = http.StatusOK
	}
	shared.WriteJSON(w, status, shared.ToAPITrip(result.Trip))
	if result.Replayed {
		h.logger.InfoContext(r.Context(), "idempotent trip replay", "trip_id", result.Trip.ID.String())
	} else {
		h.logger.InfoContext(r.Context(), "trip created", "trip_id", result.Trip.ID.String())
	}
}
