//go:generate go run go.uber.org/mock/mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package positions

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type TripSurveyRepository interface {
	GetTripsSurvey(ctx context.Context, positionStaleAfter time.Duration,
		batchSize int,
		lastTime time.Time,
		lastID uuid.UUID) ([]trip.Trip, error)
}

type pushGateway interface {
	RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error
}

type scannerMetrics interface {
	StaleCheck(ctx context.Context, result string)
	AddPositionQueueSize(ctx context.Context, delta int64)
}

type workerMetrics interface {
	PositionRequest(ctx context.Context, result string)
	RecordPositionRequestDuration(ctx context.Context, duration time.Duration)
	AddActiveWorker(ctx context.Context, delta int64)
	AddPositionQueueSize(ctx context.Context, delta int64)
}
