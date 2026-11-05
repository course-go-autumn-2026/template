package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type Usecase struct {
	tripRepo tripRepository
}

func New(tripRepo tripRepository) *Usecase {
	return &Usecase{tripRepo: tripRepo}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (*model.Trip, error) {
	return u.tripRepo.GetByID(ctx, id)
}
