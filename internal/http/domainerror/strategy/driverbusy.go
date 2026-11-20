package strategy

import (
	"errors"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type DriverBusyStrategy struct{}

func NewDriverBusyStrategy() *DriverBusyStrategy {
	return &DriverBusyStrategy{}
}

func (s *DriverBusyStrategy) Handle(err error) (int, api.Problem, bool) {
	if errors.Is(err, trip.ErrDriverBusy) {
		return http.StatusConflict, NewProblem(
			http.StatusConflict,
			"driver-busy",
			"Driver busy",
			"Driver already has an active trip",
			"driver_busy",
		), true
	}
	return 0, api.Problem{}, false
}
