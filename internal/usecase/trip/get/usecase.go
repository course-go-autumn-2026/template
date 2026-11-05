package get

import (
	"context"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type UseCase struct {
	repo TripRepository
}

func New(repo TripRepository) *UseCase {
	return &UseCase{repo: repo}
}

func (u *UseCase) Execute(ctx context.Context, id string) (*appmodel.Trip, error) {
	return u.repo.GetByID(ctx, id)
}