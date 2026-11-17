package listpositions

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/freyliee/template/internal/domain/trip"
)

type Usecase struct{ repository tripRepository }

func New(repository tripRepository) *Usecase { return &Usecase{repository: repository} }

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) ([]trip.Position, error) {
	if _, err := u.repository.GetTrip(ctx, id); err != nil {
		return nil, fmt.Errorf("get trip for route: %w", err)
	}

	positions, err := u.repository.ListPositions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get route: %w", err)
	}

	return positions, nil
}
