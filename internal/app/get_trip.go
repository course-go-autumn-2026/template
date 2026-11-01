package app

import (
	"encoding/json"
	"log"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	api "github.com/makkssik/template/internal/generated"
)

func (i *Implementation) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if err := validateGetTripRequest(tripID); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	trip, err := i.tripsService.GetTrip(r.Context(), tripID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(domainToAPI(*trip)); err != nil {
		log.Printf("write get trip response: %v", err)
	}
}

func validateGetTripRequest(tripID api.TripId) error {
	return validation.Validate(
		tripID,
		validation.Required,
		is.UUID,
	)
}
