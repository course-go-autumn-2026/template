package list

import (
	"context"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type Usecase struct {
	repo positionRepository
}

func New(repo positionRepository) *Usecase {
	return &Usecase{repo: repo}
}

func (u *Usecase) Execute(ctx context.Context, tripID uuid.UUID) ([]model.Position, error) {
	return u.repo.ListPositions(ctx, tripID)
}
