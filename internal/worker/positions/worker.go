package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type WorkerPool struct {
	gateway pushGateway
	queue   *Queue
	mu      *sync.Mutex
	wg      *sync.WaitGroup
	metrics workerMetrics
	logger  *slog.Logger
	tracer  trace.Tracer
}

func NewWorkerPool(
	gateway pushGateway,
	queue *Queue,
	mu *sync.Mutex,
	metrics workerMetrics,
	logger *slog.Logger,
) *WorkerPool {
	return &WorkerPool{
		gateway: gateway,
		queue:   queue,
		mu:      mu,
		wg:      &sync.WaitGroup{},
		metrics: metrics,
		logger:  logger,
		tracer:  otel.GetTracerProvider().Tracer("trip-service"),
	}
}

func (w *WorkerPool) Start(gCount int,
	ctx context.Context,
	timeout time.Duration,
) {
	for range gCount {
		w.wg.Go(func() {
			for {
				task, contextStatus := w.queue.Dequeue(ctx)
				if contextStatus {
					return
				}
				if task != nil {
					w.processTask(ctx, task, timeout)
				}
			}
		})
	}
}

func (w *WorkerPool) processTask(ctx context.Context, task *Task, timeout time.Duration) {
	trip := task.Trip

	if w.metrics != nil {
		w.metrics.AddActiveWorker(ctx, 1)
		w.metrics.AddPositionQueueSize(ctx, -1)
		defer w.metrics.AddActiveWorker(ctx, -1)
	}

	var spanOpts []trace.SpanStartOption
	if task.SpanContext.IsValid() {
		spanOpts = append(spanOpts, trace.WithLinks(trace.Link{SpanContext: task.SpanContext}))
	}
	workerCtx, span := w.tracer.Start(context.Background(), "positions.worker.process", spanOpts...)
	defer span.End()

	span.SetAttributes(
		attribute.String("trip_id", trip.ID.String()),
		attribute.String("driver_id", trip.DriverID.String()),
	)

	w.mu.Lock()
	if _, ok := w.queue.status[trip.ID]; !ok {
		w.mu.Unlock()
		if w.metrics != nil {
			w.metrics.PositionRequest(workerCtx, "dropped")
		}
		if w.logger != nil {
			w.logger.InfoContext(workerCtx, "position request dropped: trip already updated",
				slog.String("trip_id", trip.ID.String()),
			)
		}
		span.SetStatus(codes.Ok, "dropped")
		return
	}
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		delete(w.queue.status, trip.ID)
		w.mu.Unlock()
	}()

	closeCtx, cancel := context.WithTimeout(workerCtx, timeout)
	defer cancel()

	start := time.Now()
	err := w.gateway.RequestPosition(closeCtx, trip.DriverID, trip.ID)
	duration := time.Since(start)

	if w.metrics != nil {
		w.metrics.RecordPositionRequestDuration(workerCtx, duration)
	}

	if err != nil {
		if w.metrics != nil {
			w.metrics.PositionRequest(workerCtx, "error")
		}
		if w.logger != nil {
			w.logger.ErrorContext(workerCtx, "failed to request position",
				slog.String("trip_id", trip.ID.String()),
				slog.String("driver_id", trip.DriverID.String()),
				slog.Any("error", err),
			)
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	if w.metrics != nil {
		w.metrics.PositionRequest(workerCtx, "success")
	}
	if w.logger != nil {
		w.logger.InfoContext(workerCtx, "position requested successfully",
			slog.String("trip_id", trip.ID.String()),
			slog.String("driver_id", trip.DriverID.String()),
		)
	}
	span.SetStatus(codes.Ok, "success")
}

func (w *WorkerPool) Wait() {
	w.wg.Wait()
}
