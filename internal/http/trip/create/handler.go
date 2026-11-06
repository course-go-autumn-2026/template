package create

import (
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	api "github.com/toooriii/tori/internal/generated"
	"github.com/toooriii/tori/internal/http/domainerror"
	"github.com/toooriii/tori/internal/model"
	usecase "github.com/toooriii/tori/internal/usecase/trip/create"
)

type Handler struct {
	uc UseCase
}

func New(uc UseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "cannot read request body")
		return
	}
	req, err := decodeTripData(body)
	if err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var key *uuid.UUID
	if params.IdempotencyKey != nil {
		k := *params.IdempotencyKey
		key = &k
	}

	out, err := h.uc.Do(r.Context(), usecase.Input{
		UserID:   req.UserId,
		DriverID: req.DriverId,
		Start:    model.Point{Latitude: req.StartPoint.Latitude, Longitude: req.StartPoint.Longitude},
		End:      model.Point{Latitude: req.EndPoint.Latitude, Longitude: req.EndPoint.Longitude},
		Price:    req.Price,
		RawBody:  body,
		Key:      key,
		Now:      time.Now().UTC(),
	})
	if err != nil {
		status, p := domainerror.Problem(err)
		writeProblemBody(w, status, p)
		return
	}

	w.Header().Set("Location", "/api/v1/trips/"+out.Trip.ID.String())
	status := http.StatusCreated
	if out.AlreadyExists {
		status = http.StatusOK
	}
	writeJSON(w, status, toAPI(out.Trip))
}
