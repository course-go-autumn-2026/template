package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	api "github.com/XrystikLK/template/internal/generated"
	"github.com/XrystikLK/template/internal/http/domainerror"
	"github.com/XrystikLK/template/internal/model"
)

const isRequiredMsg = "is required"

type Handler struct {
	uc useCase
}

func New(uc useCase) *Handler {
	return &Handler{
		uc: uc,
	}
}

type rawCoordinates struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type rawCreateTripRequest struct {
	UserID     *uuid.UUID      `json:"user_id"`
	DriverID   *uuid.UUID      `json:"driver_id"`
	StartPoint *rawCoordinates `json:"start_point"`
	EndPoint   *rawCoordinates `json:"end_point"`
	Price      *int64          `json:"price"`
}

func toAPITrip(m *model.Trip) api.Trip {
	return api.Trip{
		Id:       m.ID,
		UserId:   m.UserID,
		DriverId: m.DriverID,
		StartPoint: api.Coordinates{
			Latitude:  m.StartPoint.Latitude,
			Longitude: m.StartPoint.Longitude,
		},
		EndPoint: api.Coordinates{
			Latitude:  m.EndPoint.Latitude,
			Longitude: m.EndPoint.Longitude,
		},
		Price:      m.Price,
		Status:     api.TripStatus(m.Status),
		StartedAt:  m.StartedAt,
		FinishedAt: m.FinishedAt,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var req rawCreateTripRequest
	if err := decoder.Decode(&req); err != nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{
			Field:   "body",
			Message: err.Error(),
		})
		return
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		domainerror.Write(w, r, &domainTrip.ValidationError{
			Field:   "body",
			Message: "unexpected data after JSON object",
		})
		return
	}

	if req.UserID == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "user_id", Message: isRequiredMsg})
		return
	}
	if *req.UserID == uuid.Nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "user_id", Message: "cannot be empty"})
		return
	}

	if req.DriverID == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "driver_id", Message: isRequiredMsg})
		return
	}
	if *req.DriverID == uuid.Nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "driver_id", Message: "cannot be empty"})
		return
	}

	if req.StartPoint == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "start_point", Message: isRequiredMsg})
		return
	}
	if req.StartPoint.Latitude == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "start_point.latitude", Message: isRequiredMsg})
		return
	}
	if *req.StartPoint.Latitude < -90 || *req.StartPoint.Latitude > 90 {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "start_point.latitude", Message: "must be between -90 and 90"})
		return
	}
	if req.StartPoint.Longitude == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "start_point.longitude", Message: isRequiredMsg})
		return
	}
	if *req.StartPoint.Longitude < -180 || *req.StartPoint.Longitude > 180 {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "start_point.longitude", Message: "must be between -180 and 180"})
		return
	}

	if req.EndPoint == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "end_point", Message: isRequiredMsg})
		return
	}
	if req.EndPoint.Latitude == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "end_point.latitude", Message: isRequiredMsg})
		return
	}
	if *req.EndPoint.Latitude < -90 || *req.EndPoint.Latitude > 90 {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "end_point.latitude", Message: "must be between -90 and 90"})
		return
	}
	if req.EndPoint.Longitude == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "end_point.longitude", Message: isRequiredMsg})
		return
	}
	if *req.EndPoint.Longitude < -180 || *req.EndPoint.Longitude > 180 {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "end_point.longitude", Message: "must be between -180 and 180"})
		return
	}

	if req.Price == nil {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "price", Message: isRequiredMsg})
		return
	}
	if *req.Price < 0 {
		domainerror.Write(w, r, &domainTrip.ValidationError{Field: "price", Message: "must be greater than or equal to 0"})
		return
	}

	tripModel, err := h.uc.Execute(r.Context(), toUseCaseRequestFromRaw(req))
	if err != nil {
		domainerror.Write(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Location", fmt.Sprintf("/api/v1/trips/%s", tripModel.ID.String()))
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(toAPITrip(tripModel))
}
