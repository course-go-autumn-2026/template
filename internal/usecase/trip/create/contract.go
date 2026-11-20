//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type (
	tripRepository interface {
		Save(ctx context.Context, trip trip.Trip) error
	}

	tripHistoryRepository interface {
		AddTransition(ctx context.Context, tripID uuid.UUID, fromStatus *trip.Status, toStatus trip.Status, reason *string) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metricsRecorder interface {
		TripCreated(ctx context.Context)
	}
)
