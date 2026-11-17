package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type Usecase struct{ repository tripRepository }

func New(repository tripRepository) *Usecase { return &Usecase{repository: repository} }

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (trip.Trip, error) {
	entity, err := u.repository.GetTrip(ctx, id)
	if err != nil {
		return trip.Trip{}, fmt.Errorf("get trip: %w", err)
	}

	return entity, nil
}
