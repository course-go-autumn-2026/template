package create

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	api "github.com/ggwpfu9/courseAvito/internal/generated"
	"github.com/ggwpfu9/courseAvito/internal/http/domainerror"
	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
	uc "github.com/ggwpfu9/courseAvito/internal/usecase/trip/create"
)

type Handler struct {
	uc *uc.UseCase
}

func New(uc *uc.UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}

	tripModel := appmodel.Trip{
		UserID:   req.UserId.String(),
		DriverID: req.DriverId.String(),
		Price:    req.Price,
		StartPoint: appmodel.Point{
			Latitude:  req.StartPoint.Latitude,
			Longitude: req.StartPoint.Longitude,
		},
		EndPoint: appmodel.Point{
			Latitude:  req.EndPoint.Latitude,
			Longitude: req.EndPoint.Longitude,
		},
	}

	trip, err := h.uc.Execute(r.Context(), tripModel)
	if err != nil {
		status, prob := domainerror.Problem(err)
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(prob)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+trip.ID)
	writeJSON(w, http.StatusCreated, mapTripToApi(trip))
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeProblem(w http.ResponseWriter, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Code: code, Title: http.StatusText(status), Status: int32(status), Detail: &detail,
	})
}