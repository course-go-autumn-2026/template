package positions

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/freyliee/template/internal/domain/trip"
)

type Config struct {
	Interval       time.Duration
	StaleAfter     time.Duration
	Workers        int
	QueueSize      int
	RequestTimeout time.Duration
	QueryTimeout   time.Duration
}

type task struct {
	trip   trip.StaleTrip
	source trace.SpanContext
	cancel context.CancelFunc
}

type Pool struct {
	cfg        Config
	repository repository
	gateway    gateway
	clock      clock
	observer   observer
	logger     *slog.Logger
	tracer     trace.Tracer
	queue      chan *task
	mu         sync.Mutex
	pending    map[uuid.UUID]*task
	active     atomic.Int64
	stopScan   context.CancelFunc
	abort      context.CancelFunc
	done       chan struct{}
}

func New(cfg Config, repository repository, gateway gateway, clock clock, observer observer, logger *slog.Logger, tracer trace.Tracer) *Pool {
	return &Pool{
		cfg:        cfg,
		repository: repository,
		gateway:    gateway,
		clock:      clock,
		observer:   observer,
		logger:     logger,
		tracer:     tracer,
		queue:      make(chan *task, cfg.QueueSize),
		pending:    make(map[uuid.UUID]*task),
		done:       make(chan struct{}),
	}
}

func (p *Pool) Start(ctx context.Context) {
	scanCtx, stop := context.WithCancel(ctx)
	workerCtx, abort := context.WithCancel(context.WithoutCancel(ctx))
	p.stopScan = stop
	p.abort = abort

	var wg sync.WaitGroup

	wg.Add(p.cfg.Workers + 1)

	for range p.cfg.Workers {
		go func() {
			defer wg.Done()

			p.work(workerCtx)
		}()
	}

	go func() {
		defer wg.Done()
		defer close(p.queue)

		p.scan(scanCtx)
	}()

	go func() {
		wg.Wait()
		abort()
		close(p.done)
	}()
}

func (p *Pool) Stop(ctx context.Context) error {
	p.stopScan()

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		p.abort()
		<-p.done

		return ctx.Err()
	}
}

func (p *Pool) Cancel(id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if task := p.pending[id]; task != nil {
		if task.cancel != nil {
			task.cancel()
		}

		delete(p.pending, id)
	}
}

func (p *Pool) Snapshot() (int64, int64) { return int64(len(p.queue)), p.active.Load() }

func (p *Pool) scan(ctx context.Context) {
	ticker := p.clock.NewTicker(p.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.Chan():
			if ctx.Err() != nil {
				return
			}

			p.check(ctx)
		}
	}
}

func (p *Pool) check(ctx context.Context) {
	ctx, span := p.tracer.Start(ctx, "positions.check")
	defer span.End()

	queryCtx, cancel := context.WithTimeout(ctx, p.cfg.QueryTimeout)
	trips, err := p.repository.FindStaleTrips(queryCtx, p.cfg.StaleAfter)
	cancel()

	if err != nil {
		if ctx.Err() == nil {
			p.logger.WarnContext(ctx, "stale position scan failed", "error", err)
		}

		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
		span.SetStatus(codes.Error, "stale position scan failed")

		return
	}

	span.SetAttributes(attribute.Int("tripgo.stale_rows", len(trips)))

	for _, entity := range trips {
		if ctx.Err() != nil {
			return
		}

		result := p.enqueue(entity, span.SpanContext())
		p.observer.StaleCheck(ctx, result)

		if result == "dropped" {
			p.observer.PositionRequest(ctx, result, 0)
			p.logger.WarnContext(ctx, "position request queue full", "trip_id", entity.ID)
		}
	}
}

func (p *Pool) enqueue(entity trip.StaleTrip, source trace.SpanContext) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.pending[entity.ID]; ok {
		return "duplicate"
	}

	task := &task{trip: entity, source: source}
	p.pending[entity.ID] = task

	select {
	case p.queue <- task:
		return "queued"
	default:
		delete(p.pending, entity.ID)

		return "dropped"
	}
}

func (p *Pool) work(ctx context.Context) {
	for task := range p.queue {
		if ctx.Err() != nil {
			p.finish(task)
			continue
		}

		p.process(ctx, task)
	}
}

func (p *Pool) process(ctx context.Context, task *task) {
	ctx, cancel := context.WithTimeout(ctx, p.cfg.RequestTimeout)
	defer cancel()

	p.mu.Lock()

	if p.pending[task.trip.ID] != task {
		p.mu.Unlock()

		return
	}

	task.cancel = cancel
	p.mu.Unlock()
	defer p.finish(task)

	ctx, span := p.tracer.Start(
		ctx,
		"positions.request",
		trace.WithNewRoot(),
		trace.WithLinks(trace.Link{SpanContext: task.source}),
	)
	defer span.End()

	p.active.Add(1)
	defer p.active.Add(-1)

	needed, err := p.repository.NeedsPosition(ctx, task.trip.ID, p.cfg.StaleAfter)

	if err == nil && (!needed || ctx.Err() != nil) {
		return
	}

	started := p.clock.Now()

	if err == nil {
		err = p.gateway.RequestPosition(ctx, task.trip.ID, task.trip.DriverID)
	}

	result := "success"

	if err != nil {
		result = "error"
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", err)))
		span.SetStatus(codes.Error, "position request failed")

		if !errors.Is(err, context.Canceled) {
			p.logger.WarnContext(ctx, "position request failed", "trip_id", task.trip.ID, "error", err)
		}
	}

	p.observer.PositionRequest(ctx, result, p.clock.Now().Sub(started))
}

func (p *Pool) finish(task *task) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.pending[task.trip.ID] == task {
		delete(p.pending, task.trip.ID)
	}
}
