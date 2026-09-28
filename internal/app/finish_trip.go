package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	api "github.com/makkssik/template/internal/generated"
)

func (i *Implementation) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	if err := validateFinishTripRequest(r, tripID); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error())
		return
	}

	trip, err := i.tripsService.FinishTrip(r.Context(), tripID)
	if err != nil {
		writeServiceError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(domainToAPI(*trip)); err != nil {
		log.Printf("write finish trip response: %v", err)
	}
}

func validateFinishTripRequest(r *http.Request, tripID api.TripId) error {
	if err := validation.Validate(tripID, validation.Required, is.UUID); err != nil {
		return err
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1))
	if err != nil {
		return fmt.Errorf("read finish trip request: %w", err)
	}
	if len(body) != 0 {
		return errors.New("request body must be empty")
	}
	return nil
}
