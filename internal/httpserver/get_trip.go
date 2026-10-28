package httpserver

import (
	"errors"
	"log"
	"net/http"

	api "github.com/Yolanesso/avito-labs/internal/generated"
	"github.com/Yolanesso/avito-labs/internal/model"
)

func (h *Handler) GetTrip(
	w http.ResponseWriter,
	r *http.Request,
	tripID api.TripId,
) {
	trip, err := h.tripService.GetByID(
		r.Context(),
		tripID,
	)

	if errors.Is(err, model.ErrTripNotFound) {
		writeProblem(
			w,
			r,
			http.StatusNotFound,
			"https://tripgo.example/problems/trip-not-found",
			"Trip not found",
			"Trip with this ID does not exist",
			"trip_not_found",
		)
		return
	}

	if err != nil {
		log.Printf("get trip: %v", err)

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

	writeJSON(
		w,
		http.StatusOK,
		response,
	)
}
