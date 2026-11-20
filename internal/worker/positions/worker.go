package positions

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Task struct {
	ID, Driver uuid.UUID
	At         time.Time
	Span       trace.SpanContext
}
type Gateway interface {
	Request(context.Context, uuid.UUID, uuid.UUID) error
}
type Ticker interface {
	Chan() <-chan time.Time
	Stop()
}
type realTicker struct{ *time.Ticker }

func (t realTicker) Chan() <-chan time.Time { return t.C }

type Pool struct {
	db                                  *pgxpool.Pool
	gateway                             Gateway
	logger                              *slog.Logger
	tracer                              trace.Tracer
	interval, stale, timeout, dbTimeout time.Duration
	batch, capacity, workers            int
	mu                                  sync.Mutex
	cond                                *sync.Cond
	queue                               []Task
	pending                             map[uuid.UUID]bool
	closing                             bool
	cancel                              context.CancelFunc
	workerCancel                        context.CancelFunc
	scanDone                            chan struct{}
	wg                                  sync.WaitGroup
	checks, requests                    metric.Int64Counter
	size, active                        metric.Int64UpDownCounter
	duration, rows                      metric.Float64Histogram
}

func New(db *pgxpool.Pool, gateway Gateway, logger *slog.Logger, meter metric.Meter, tracer trace.Tracer, interval, stale, timeout, dbTimeout time.Duration, batch, capacity, workers int) (*Pool, error) {
	p := &Pool{db: db, gateway: gateway, logger: logger, tracer: tracer, interval: interval, stale: stale, timeout: timeout, dbTimeout: dbTimeout, batch: batch, capacity: capacity, workers: workers, pending: make(map[uuid.UUID]bool), scanDone: make(chan struct{})}
	p.cond = sync.NewCond(&p.mu)
	var err error
	if p.checks, err = meter.Int64Counter("tripgo.stale_checks"); err != nil {
		return nil, err
	}
	if p.requests, err = meter.Int64Counter("tripgo.position_requests"); err != nil {
		return nil, err
	}
	if p.size, err = meter.Int64UpDownCounter("tripgo.position_requests.queue.size"); err != nil {
		return nil, err
	}
	if p.active, err = meter.Int64UpDownCounter("tripgo.position_requests.workers.active"); err != nil {
		return nil, err
	}
	if p.duration, err = meter.Float64Histogram("tripgo.position_requests.duration", metric.WithUnit("s")); err != nil {
		return nil, err
	}
	if p.rows, err = meter.Float64Histogram("tripgo.stale_checks.rows_read"); err != nil {
		return nil, err
	}
	return p, nil
}
func (p *Pool) Start(ctx context.Context) { p.start(ctx, realTicker{time.NewTicker(p.interval)}) }
func (p *Pool) start(ctx context.Context, ticker Ticker) {
	scanCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	workerCtx, workerCancel := context.WithCancel(ctx)
	p.workerCancel = workerCancel
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.work(workerCtx)
	}
	go func() {
		defer close(p.scanDone)
		defer func() { p.mu.Lock(); p.closing = true; p.cond.Broadcast(); p.mu.Unlock() }()
		defer ticker.Stop()
		for {
			select {
			case <-scanCtx.Done():
				return
			case <-ticker.Chan():
				p.scan(scanCtx)
			}
		}
	}()
}

func (p *Pool) scan(ctx context.Context) {
	ctx, span := p.tracer.Start(ctx, "positions.scan")
	defer span.End()
	var at time.Time
	var id uuid.UUID
	first := true
	count := 0
	defer func() { p.rows.Record(ctx, float64(count)) }()
	for {
		qctx, cancel := context.WithTimeout(ctx, p.dbTimeout)
		rows, err := p.db.Query(qctx, `SELECT id, driver_id, COALESCE(last_position_at, started_at) AS stale_at FROM trips WHERE status = 'active' AND COALESCE(last_position_at, started_at) < now() - ($1::double precision * interval '1 second') AND ($2::boolean OR (COALESCE(last_position_at, started_at), id) > ($3::timestamptz, $4::uuid)) ORDER BY COALESCE(last_position_at, started_at), id LIMIT $5`, p.stale.Seconds(), first, at, id, p.batch)
		if err != nil {
			cancel()
			if ctx.Err() == nil {
				p.logger.ErrorContext(ctx, "scan stale trips", "error", err)
			}
			return
		}
		tasks := make([]Task, 0, p.batch)
		for rows.Next() {
			var t Task
			if err = rows.Scan(&t.ID, &t.Driver, &t.At); err != nil {
				break
			}
			t.Span = span.SpanContext()
			tasks = append(tasks, t)
		}
		if err == nil {
			err = rows.Err()
		}
		rows.Close()
		cancel()
		if err != nil {
			p.logger.ErrorContext(ctx, "read stale trips", "error", err)
			return
		}
		count += len(tasks)
		for _, t := range tasks {
			if !p.enqueue(ctx, t) {
				return
			}
		}
		if len(tasks) < p.batch {
			return
		}
		last := tasks[len(tasks)-1]
		at, id, first = last.At, last.ID, false
	}
}

func (p *Pool) enqueue(ctx context.Context, t Task) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := "queued"
	dropped := false
	switch {
	case p.pending[t.ID]:
		result = "duplicate"
	case len(p.queue) == p.capacity:
		newest := 0
		for i := 1; i < len(p.queue); i++ {
			if older(p.queue[newest], p.queue[i]) {
				newest = i
			}
		}
		if older(t, p.queue[newest]) {
			delete(p.pending, p.queue[newest].ID)
			p.queue[newest] = t
			p.pending[t.ID] = true
			p.checks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "dropped")))
			p.logger.WarnContext(ctx, "position task displaced")
			dropped = true
		} else {
			result = "dropped"
			dropped = true
			p.logger.WarnContext(ctx, "position queue full")
		}
	default:
		p.queue = append(p.queue, t)
		p.pending[t.ID] = true
		p.size.Add(ctx, 1)
		p.cond.Signal()
	}
	p.checks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	return !dropped
}

func older(a, b Task) bool {
	if a.At.Equal(b.At) {
		return a.ID.String() < b.ID.String()
	}
	return a.At.Before(b.At)
}

func (p *Pool) Cancel(ctx context.Context, id uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, t := range p.queue {
		if t.ID == id {
			p.queue = append(p.queue[:i], p.queue[i+1:]...)
			delete(p.pending, id)
			p.size.Add(ctx, -1)
			p.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", "cancelled")))
			return
		}
	}
}

func (p *Pool) work(ctx context.Context) {
	defer p.wg.Done()
	for {
		p.mu.Lock()
		for len(p.queue) == 0 && !p.closing {
			p.cond.Wait()
		}
		if len(p.queue) == 0 && p.closing {
			p.mu.Unlock()
			return
		}
		oldest := 0
		for i := 1; i < len(p.queue); i++ {
			if older(p.queue[i], p.queue[oldest]) {
				oldest = i
			}
		}
		t := p.queue[oldest]
		p.queue = append(p.queue[:oldest], p.queue[oldest+1:]...)
		p.size.Add(ctx, -1)
		p.mu.Unlock()
		p.active.Add(ctx, 1)
		taskCtx, span := p.tracer.Start(ctx, "positions.request", trace.WithLinks(trace.Link{SpanContext: t.Span}))
		callCtx, cancel := context.WithTimeout(taskCtx, p.timeout)
		start := time.Now()
		err := p.gateway.Request(callCtx, t.ID, t.Driver)
		cancel()
		result := "success"
		if err != nil {
			result = "error"
			p.logger.WarnContext(taskCtx, "position push failed", "error", err)
		}
		p.requests.Add(taskCtx, 1, metric.WithAttributes(attribute.String("result", result)))
		p.duration.Record(taskCtx, time.Since(start).Seconds(), metric.WithAttributes(attribute.String("result", result)))
		span.End()
		p.active.Add(ctx, -1)
		p.mu.Lock()
		delete(p.pending, t.ID)
		p.mu.Unlock()
	}
}

func (p *Pool) Stop(ctx context.Context) error {
	p.cancel()
	select {
	case <-p.scanDone:
	case <-ctx.Done():
		p.workerCancel()
		p.mu.Lock()
		p.closing = true
		p.cond.Broadcast()
		p.mu.Unlock()
		return fmt.Errorf("stop scanner: %w", ctx.Err())
	}
	p.mu.Lock()
	p.closing = true
	p.cond.Broadcast()
	p.mu.Unlock()
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
		p.workerCancel()
		return nil
	case <-ctx.Done():
		p.workerCancel()
		return fmt.Errorf("drain workers: %w", ctx.Err())
	}
}
