package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/XrystikLK/template/internal/model"
)

type UseCase struct {
	repo tripRepository
}

func New(repo tripRepository) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	return uc.repo.GetByID(ctx, id)
}
