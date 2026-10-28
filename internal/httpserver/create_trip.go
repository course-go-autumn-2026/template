package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"

	api "github.com/Yolanesso/avito-labs/internal/generated"
	"github.com/Yolanesso/avito-labs/internal/model"
	"github.com/Yolanesso/avito-labs/internal/service"
)

func (h *Handler) CreateTrip(
	w http.ResponseWriter,
	r *http.Request,
	_ api.CreateTripParams,
) {
	var request api.CreateTripJSONRequestBody

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&request)
	if err != nil {
		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"https://tripgo.example/problems/invalid-request",
			"Invalid request",
			fmt.Sprintf("Invalid JSON body: %v", err),
			"invalid_request",
		)
		return
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		writeProblem(
			w,
			r,
			http.StatusBadRequest,
			"https://tripgo.example/problems/invalid-request",
			"Invalid request",
			"Request body must contain one JSON object",
			"invalid_request",
		)
		return
	}

	if request.UserId == uuid.Nil {
		writeInvalidRequest(
			w,
			r,
			"user_id must not be empty",
		)
		return
	}

	if request.DriverId == uuid.Nil {
		writeInvalidRequest(
			w,
			r,
			"driver_id must not be empty",
		)
		return
	}

	if !coordinatesAreValid(request.StartPoint) {
		writeInvalidRequest(
			w,
			r,
			"start_point contains invalid coordinates",
		)
		return
	}

	if !coordinatesAreValid(request.EndPoint) {
		writeInvalidRequest(
			w,
			r,
			"end_point contains invalid coordinates",
		)
		return
	}

	if request.Price < 0 {
		writeInvalidRequest(
			w,
			r,
			"price must not be negative",
		)
		return
	}

	input := service.CreateTripInput{
		UserID:   request.UserId,
		DriverID: request.DriverId,

		StartPoint: model.Coordinates{
			Latitude:  request.StartPoint.Latitude,
			Longitude: request.StartPoint.Longitude,
		},

		EndPoint: model.Coordinates{
			Latitude:  request.EndPoint.Latitude,
			Longitude: request.EndPoint.Longitude,
		},

		Price: request.Price,
	}

	trip, err := h.tripService.Create(
		r.Context(),
		input,
	)

	if errors.Is(err, model.ErrDriverBusy) {
		writeProblem(
			w,
			r,
			http.StatusConflict,
			"https://tripgo.example/problems/driver-busy",
			"Driver busy",
			"Driver already has an active trip",
			"driver_busy",
		)
		return
	}

	if err != nil {
		log.Printf("create trip: %v", err)

		writeProblem(
			w,
			r,
			http.StatusInternalServerError,
			"https://tripgo.example/problems/internal-error",
			"Internal error",
			"Internal server error",
			"internal_error",
		)
		return
	}

	response := tripToAPI(trip)

	w.Header().Set(
		"Location",
		"/api/v1/trips/"+trip.ID.String(),
	)

	writeJSON(
		w,
		http.StatusCreated,
		response,
	)
}

func coordinatesAreValid(coordinates api.Coordinates) bool {
	if coordinates.Latitude < -90 {
		return false
	}

	if coordinates.Latitude > 90 {
		return false
	}

	if coordinates.Longitude < -180 {
		return false
	}

	if coordinates.Longitude > 180 {
		return false
	}

	return true
}

func writeInvalidRequest(
	w http.ResponseWriter,
	r *http.Request,
	detail string,
) {
	writeProblem(
		w,
		r,
		http.StatusBadRequest,
		"https://tripgo.example/problems/invalid-request",
		"Invalid request",
		detail,
		"invalid_request",
	)
}
