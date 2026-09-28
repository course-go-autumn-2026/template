package trips

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	history "github.com/makkssik/template/internal/domain/trip_status_history"
	domaintrips "github.com/makkssik/template/internal/domain/trips"
	"github.com/samber/lo"
)

func (s *Service) FinishTrip(ctx context.Context, tripID uuid.UUID) (*domaintrips.Trip, error) {
	var trip *domaintrips.Trip
	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		var err error
		trip, err = s.tripsRepository.FinishTrip(txCtx, domaintrips.FinishTripDTO{
			TripID:     tripID,
			FinishedAt: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		return s.statusHistoryRepository.RegisterStatusChange(txCtx, history.StatusChange{
			TripID:     trip.TripID,
			FromStatus: lo.ToPtr(domaintrips.StatusActive),
			ToStatus:   domaintrips.StatusCompleted,
			ChangedAt:  *trip.FinishedAt,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("finish trip: %w", err)
	}
	return trip, nil
}
