//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package get

import (
	"context"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type (
	tripPositionRepository interface {
		GetPositions(ctx context.Context, id uuid.UUID) ([]trip.Position, error)
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	tripRepository interface {
		GetTrip(ctx context.Context, id uuid.UUID) (trip.Trip, error)
	}
)
