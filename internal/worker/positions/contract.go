//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type (
	staleTripRepository interface {
		FindStaleActiveTrips(ctx context.Context, staleAfter time.Duration) ([]model.StaleTrip, error)
	}

	pushGateway interface {
		RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
	}

	Ticker interface {
		C() <-chan time.Time
		Stop()
	}
)
