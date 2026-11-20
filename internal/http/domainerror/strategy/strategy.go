package strategy

import (
	api "github.com/George1487/MalyginGeorge/internal/generated"
)

type ErrorStrategy interface {
	Handle(err error) (int, api.Problem, bool)
}

func NewProblem(status int, problemType, title, detail, code string) api.Problem {
	return api.Problem{
		Type:   "https://tripgo.example/problems/" + problemType,
		Title:  title,
		Status: int32(status), // #nosec G115 -- HTTP status codes fit within int32
		Detail: &detail,
		Code:   code,
	}
}

func DefaultStrategies() []ErrorStrategy {
	return []ErrorStrategy{
		NewValidationStrategy(),
		NewNotFoundStrategy(),
		NewCompletedStrategy(),
		NewDriverBusyStrategy(),
		NewIdempotencyStrategy(),
	}
}
