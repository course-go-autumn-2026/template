package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/toooriii/tori/internal/model"
)

type UseCase struct {
	trips tripRepository
}

func New(trips tripRepository) *UseCase {
	return &UseCase{trips: trips}
}

func (uc *UseCase) Do(ctx context.Context, id uuid.UUID) (model.Trip, error) {
	return uc.trips.Get(ctx, id)
}
