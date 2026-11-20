package strategy

import (
	"errors"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type IdempotencyStrategy struct{}

func NewIdempotencyStrategy() *IdempotencyStrategy {
	return &IdempotencyStrategy{}
}

func (s *IdempotencyStrategy) Handle(err error) (int, api.Problem, bool) {
	if errors.Is(err, trip.ErrIdempotencyConflict) {
		return http.StatusConflict, NewProblem(
			http.StatusConflict,
			"idempotency-conflict",
			"Idempotency conflict",
			"Idempotency-Key was already used with a different request body",
			"idempotency_conflict",
		), true
	}
	return 0, api.Problem{}, false
}
