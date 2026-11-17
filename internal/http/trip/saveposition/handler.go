package saveposition

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
	api "github.com/freyliee/template/internal/generated"
	"github.com/freyliee/template/internal/http/jsonbody"
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
	id, err := uuid.Parse(chi.URLParam(r, "tripId"))
	if err != nil {
		respond.Invalid(w, r, h.logger, "tripId must be a UUID")

		return
	}

	var data api.CreateTripPositionJSONRequestBody

	if err := jsonbody.Decode(
		r.Body,
		&data,
		jsonbody.Shape{"coordinates": {"latitude": nil, "longitude": nil}, "recorded_at": nil},
	); err != nil {
		respond.Invalid(w, r, h.logger, err.Error())

		return
	}

	entity, err := h.usecase.Execute(
		r.Context(),
		id,
		trip.PositionData{
			Coordinates: trip.Coordinates{Latitude: data.Coordinates.Latitude, Longitude: data.Coordinates.Longitude},
			RecordedAt:  data.RecordedAt,
		},
	)
	if err != nil {
		respond.Error(w, r, h.logger, err)

		return
	}

	respond.JSON(w, r, h.logger, http.StatusCreated, "application/json", convert.Position(entity))
}
