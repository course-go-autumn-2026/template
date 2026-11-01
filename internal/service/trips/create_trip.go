package trips

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	history "github.com/makkssik/template/internal/domain/trip_status_history"
	domaintrips "github.com/makkssik/template/internal/domain/trips"
)

func (s *Service) CreateTrip(ctx context.Context, opts CreateTripOpts) (*domaintrips.Trip, error) {
	now := time.Now().UTC()
	trip := domaintrips.Trip{
		TripID:     uuid.New(),
		UserID:     opts.UserID,
		DriverID:   opts.DriverID,
		StartPoint: opts.StartPoint,
		EndPoint:   opts.EndPoint,
		Price:      opts.Price,
		Status:     domaintrips.StatusActive,
		StartedAt:  now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := s.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := s.tripsRepository.CreateTrip(txCtx, trip); err != nil {
			return err
		}
		return s.statusHistoryRepository.RegisterStatusChange(txCtx, history.StatusChange{
			TripID:    trip.TripID,
			ToStatus:  domaintrips.StatusActive,
			ChangedAt: trip.StartedAt,
		})
	})
	if err != nil {
		return nil, fmt.Errorf("create trip: %w", err)
	}
	return &trip, nil
}
