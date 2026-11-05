package finish

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	api "github.com/ggwpfu9/courseAvito/internal/generated"
	"github.com/ggwpfu9/courseAvito/internal/http/domainerror"
	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	uc "github.com/ggwpfu9/courseAvito/internal/usecase/trip/finish"
)

type Handler struct {
	uc *uc.UseCase
}

func New(uc *uc.UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripId")

	trip, err := h.uc.Execute(r.Context(), tripID)
	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(mapTripToApi(trip))
}

func mapTripToApi(t *appmodel.Trip) api.Trip {
	id, _ := uuid.Parse(t.ID)
	userID, _ := uuid.Parse(t.UserID)
	driverID, _ := uuid.Parse(t.DriverID)

	return api.Trip{
		Id:         id,
		UserId:     userID,
		DriverId:   driverID,
		Price:      t.Price,
		Status:     api.TripStatus(t.Status),
		StartedAt:  t.StartedAt,
		FinishedAt: t.FinishedAt,
		StartPoint: api.Coordinates{Latitude: t.StartPoint.Latitude, Longitude: t.StartPoint.Longitude},
		EndPoint:   api.Coordinates{Latitude: t.EndPoint.Latitude, Longitude: t.EndPoint.Longitude},
	}
}