package strategy

import (
	"errors"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type CompletedStrategy struct{}

func NewCompletedStrategy() *CompletedStrategy {
	return &CompletedStrategy{}
}

func (s *CompletedStrategy) Handle(err error) (int, api.Problem, bool) {
	if errors.Is(err, trip.ErrTripCompleted) {
		return http.StatusConflict, NewProblem(
			http.StatusConflict,
			"trip-completed",
			"Trip completed",
			"Operation is not allowed for a completed trip",
			"trip_completed",
		), true
	}
	return 0, api.Problem{}, false
}
