package strategy

import (
	"errors"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type NotFoundStrategy struct{}

func NewNotFoundStrategy() *NotFoundStrategy {
	return &NotFoundStrategy{}
}

func (s *NotFoundStrategy) Handle(err error) (int, api.Problem, bool) {
	if errors.Is(err, trip.ErrTripNotFound) {
		return http.StatusNotFound, NewProblem(
			http.StatusNotFound,
			"trip-not-found",
			"Trip not found",
			"Trip was not found",
			"trip_not_found",
		), true
	}
	return 0, api.Problem{}, false
}
