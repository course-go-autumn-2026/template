package strategy

import (
	"errors"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type ValidationStrategy struct{}

func NewValidationStrategy() *ValidationStrategy {
	return &ValidationStrategy{}
}

func (s *ValidationStrategy) Handle(err error) (int, api.Problem, bool) {
	var validationErr *trip.ValidationError
	if errors.As(err, &validationErr) || errors.Is(err, trip.ErrInvalidRequest) {
		detail := "Request validation failed"
		if validationErr != nil && validationErr.Message != "" {
			detail = validationErr.Error()
		}
		return http.StatusBadRequest, NewProblem(
			http.StatusBadRequest,
			"invalid-request",
			"Invalid request",
			detail,
			"invalid_request",
		), true
	}
	return 0, api.Problem{}, false
}
