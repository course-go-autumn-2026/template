//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type (
	tripRepository interface {
		GetTripForUpdate(ctx context.Context, id uuid.UUID) (trip.Trip, error)
		Update(ctx context.Context, t trip.Trip) error
	}

	tripHistoryRepository interface {
		AddTransition(ctx context.Context, tripID uuid.UUID, fromStatus *trip.Status, toStatus trip.Status, reason *string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metricsRecorder interface {
		TripCompleted(ctx context.Context)
	}
)
