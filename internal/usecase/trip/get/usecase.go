package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type Usecase struct {
	tripRepo tripRepository
}

func New(tripRepo tripRepository) *Usecase {
	return &Usecase{tripRepo: tripRepo}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	foundTrip, err := u.tripRepo.GetTrip(ctx, id)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return foundTrip, nil
}
