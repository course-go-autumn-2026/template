package positions

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	staleCheckResultQueued    = "queued"
	staleCheckResultDuplicate = "duplicate"
	staleCheckResultDropped   = "dropped"
)

type Scaner struct {
	tripRepo TripSurveyRepository
	ticker   clockwork.Ticker
	metrics  scannerMetrics
	logger   *slog.Logger
	queue    *Queue
	tracer   trace.Tracer
}

func New(tripRepo TripSurveyRepository,
	ticker clockwork.Ticker,
	metrics scannerMetrics,
	logger *slog.Logger,
	queue *Queue,
) *Scaner {
	return &Scaner{
		tripRepo: tripRepo,
		ticker:   ticker,
		metrics:  metrics,
		logger:   logger,
		queue:    queue,
		tracer:   otel.GetTracerProvider().Tracer("trip-service"),
	}
}

func (s *Scaner) Run(ctx context.Context,
	positionCheckInterval time.Duration,
	positionStaleAfter time.Duration,
	batchSize int,
) error {
	batch_iterations := (cap(s.queue.trips) + batchSize - 1) / batchSize
	s.ticker.Reset(positionCheckInterval)
loop:
	for {
		select {
		case <-ctx.Done():
			s.ticker.Stop()
			s.queue.Close()
			return ctx.Err()
		case <-s.ticker.Chan():
			last_id := uuid.Nil
			last_time := time.Unix(0, 0)
			scanCtx, scanSpan := s.tracer.Start(ctx, "positions.scanner.check")
			for range batch_iterations {
				trips, err := s.tripRepo.GetTripsSurvey(scanCtx, positionStaleAfter, batchSize, last_time, last_id)
				if err != nil {
					if s.logger != nil {
						s.logger.ErrorContext(scanCtx, "failed to get stale trips", slog.Any("error", err))
					}
					scanSpan.RecordError(err)
					scanSpan.SetStatus(codes.Error, err.Error())
					scanSpan.End()
					continue loop
				}
				for i, trip := range trips {
					var result string
					s.queue.Enqueue(trip, scanSpan.SpanContext(), &result)
					if s.metrics != nil {
						s.metrics.StaleCheck(scanCtx, result)
						if result == staleCheckResultQueued {
							s.metrics.AddPositionQueueSize(scanCtx, 1)
						}
					}
					if result == staleCheckResultDropped {
						if s.logger != nil {
							s.logger.WarnContext(scanCtx, "position request queue is full, trip dropped", slog.String("trip_id", trip.ID.String()))
						}
						scanSpan.SetStatus(codes.Ok, "queue full")
						scanSpan.End()
						continue loop
					}
					if i == len(trips)-1 {
						last_id = trip.ID
						if trip.LastPositionAt != nil {
							last_time = *trip.LastPositionAt
							continue
						}
						last_time = trip.StartedAt
					}
				}
				if len(trips) < batchSize {
					scanSpan.SetStatus(codes.Ok, "completed")
					scanSpan.End()
					continue loop
				}
			}
			scanSpan.SetStatus(codes.Ok, "completed")
			scanSpan.End()
		}
	}
}
