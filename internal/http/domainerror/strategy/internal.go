package strategy

import (
	"net/http"

	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type InternalStrategy struct{}

func NewInternalStrategy() *InternalStrategy {
	return &InternalStrategy{}
}

func (s *InternalStrategy) Handle(_ error) (int, api.Problem, bool) {
	return http.StatusInternalServerError, NewProblem(
		http.StatusInternalServerError,
		"internal-error",
		"Internal Server Error",
		"Internal server error",
		"internal_error",
	), true
}
