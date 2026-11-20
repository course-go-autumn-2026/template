//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type (
	tripRepository interface {
		GetTrip(ctx context.Context, id uuid.UUID) (trip.Trip, error)
	}

	tripPositionRepository interface {
		AddPosition(ctx context.Context, position *trip.Position) (*trip.Position, error)
		UpdateLastPositionAt(ctx context.Context, tripID uuid.UUID, LastPositionAt time.Time) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metricsRecorder interface {
		PositionSaved(ctx context.Context)
	}
)
