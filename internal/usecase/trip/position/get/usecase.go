package get

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type Usecase struct {
	positionRepo tripPositionRepository
	txManager    txManager
	tripRepo     tripRepository
}

func NewUsecase(positionRepo tripPositionRepository,
	txManager txManager,
	tripRepo tripRepository,
) *Usecase {
	return &Usecase{
		positionRepo: positionRepo,
		txManager:    txManager,
		tripRepo:     tripRepo,
	}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) ([]trip.Position, error) {
	var positions []trip.Position
	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		trip, err := u.tripRepo.GetTrip(txCtx, id)
		if err != nil {
			return fmt.Errorf("failed to get trip for update: %w", err)
		}
		positions, err = u.positionRepo.GetPositions(txCtx, trip.ID)
		if err != nil {
			return fmt.Errorf("failed to get positions: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}
	return positions, nil
}
