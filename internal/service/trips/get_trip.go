package trips

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domaintrips "github.com/makkssik/template/internal/domain/trips"
)

func (s *Service) GetTrip(ctx context.Context, tripID uuid.UUID) (*domaintrips.Trip, error) {
	trip, err := s.tripsRepository.GetTripByTripId(ctx, tripID)
	if err != nil {
		return nil, fmt.Errorf("get trip: %w", err)
	}
	return trip, nil
}
