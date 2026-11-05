package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/model"
)

type Usecase struct {
	repo tripRepository
}

func New(repo tripRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	if id == uuid.Nil {
		return nil, trip.NewValidationError("id", "trip id cannot be empty")
	}

	return u.repo.GetByID(ctx, id)
}
