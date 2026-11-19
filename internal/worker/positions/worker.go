package positions

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/xotapy-main/courseGo/internal/model"
)

const (
	resultQueued    = "queued"
	resultDuplicate = "duplicate"
	resultDropped   = "dropped"
	resultOK        = "ok"
	resultError     = "error"
	resultCancelled = "cancelled"
)

type Config struct {
	CheckInterval  time.Duration
	StaleAfter     time.Duration
	Workers        int
	QueueSize      int
	RequestTimeout time.Duration
}

type Metrics struct {
	StaleChecks     metric.Int64Counter
	QueueSize       metric.Int64UpDownCounter
	WorkersActive   metric.Int64UpDownCounter
	Requests        metric.Int64Counter
	RequestDuration metric.Float64Histogram
}

type pollTask struct {
	tripID      uuid.UUID
	driverID    uuid.UUID
	seq         uint64
	scanSpanCtx trace.SpanContext
}

type Poller struct {
	repo    staleTripRepository
	pushGW  pushGateway
	ticker  Ticker
	logger  *slog.Logger
	metrics Metrics
	cfg     Config
	tracer  trace.Tracer

	queue    chan pollTask
	mu       sync.Mutex
	inFlight map[uuid.UUID]uint64
	seq      uint64

	scannerCancel context.CancelFunc
	workersCancel context.CancelFunc
	scannerWG     sync.WaitGroup
	workersWG     sync.WaitGroup
	stopOnce      sync.Once
}

func New(
	repo staleTripRepository,
	pushGW pushGateway,
	ticker Ticker,
	logger *slog.Logger,
	metrics Metrics,
	cfg Config,
) *Poller {
	if ticker == nil {
		ticker = NewRealTicker(cfg.CheckInterval)
	}

	return &Poller{
		repo:     repo,
		pushGW:   pushGW,
		ticker:   ticker,
		logger:   logger,
		metrics:  metrics,
		cfg:      cfg,
		tracer:   otel.Tracer("trip-service"),
		queue:    make(chan pollTask, cfg.QueueSize),
		inFlight: make(map[uuid.UUID]uint64, cfg.QueueSize),
	}
}

func (p *Poller) Start(ctx context.Context) {
	scannerCtx, scannerCancel := context.WithCancel(ctx)
	workersCtx, workersCancel := context.WithCancel(context.Background())

	p.scannerCancel = scannerCancel
	p.workersCancel = workersCancel

	for range p.cfg.Workers {
		p.workersWG.Add(1)
		go p.runWorker(workersCtx)
	}

	p.scannerWG.Add(1)
	go p.runScanner(scannerCtx)
}

func (p *Poller) Cancel(tripID uuid.UUID) {
	p.mu.Lock()
	delete(p.inFlight, tripID)
	p.mu.Unlock()
}

func (p *Poller) Shutdown(ctx context.Context) error {
	p.stopOnce.Do(func() {
		if p.scannerCancel != nil {
			p.scannerCancel()
		}
	})

	p.scannerWG.Wait()

	workersDone := make(chan struct{})
	go func() {
		p.workersWG.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		if p.workersCancel != nil {
			p.workersCancel()
		}
		return nil
	case <-ctx.Done():
		if p.workersCancel != nil {
			p.workersCancel()
		}
		<-workersDone
		return ctx.Err()
	}
}

func (p *Poller) runScanner(ctx context.Context) {
	defer p.scannerWG.Done()
	defer close(p.queue)
	defer p.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.ticker.C():
			p.scanOnce(ctx)
		}
	}
}

func (p *Poller) scanOnce(ctx context.Context) {
	scanCtx, span := p.tracer.Start(ctx, "positions.scan")
	defer span.End()

	trips, err := p.repo.FindStaleActiveTrips(scanCtx, p.cfg.StaleAfter)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "db_error"))
		p.logger.ErrorContext(scanCtx, "failed to query stale trips", slog.String("error", err.Error()))
		return
	}

	spanCtx := span.SpanContext()
	for _, t := range trips {
		if ctx.Err() != nil {
			return
		}
		p.enqueue(scanCtx, t, spanCtx)
	}
}

func (p *Poller) enqueue(ctx context.Context, t model.StaleTrip, scanSpanCtx trace.SpanContext) {
	p.mu.Lock()
	if _, exists := p.inFlight[t.ID]; exists {
		p.mu.Unlock()
		p.recordStaleCheck(ctx, resultDuplicate)
		return
	}

	p.seq++
	task := pollTask{
		tripID:      t.ID,
		driverID:    t.DriverID,
		seq:         p.seq,
		scanSpanCtx: scanSpanCtx,
	}

	select {
	case p.queue <- task:
		p.inFlight[t.ID] = task.seq
		p.mu.Unlock()
		p.recordQueueSize(ctx, 1)
		p.recordStaleCheck(ctx, resultQueued)
	default:
		p.mu.Unlock()
		p.recordStaleCheck(ctx, resultDropped)
		p.recordPositionRequest(ctx, resultDropped)
		p.logger.WarnContext(ctx, "position poll queue is full, task dropped",
			slog.String("trip_id", t.ID.String()),
		)
	}
}

func (p *Poller) runWorker(ctx context.Context) {
	defer p.workersWG.Done()

	for task := range p.queue {
		p.recordQueueSize(ctx, -1)
		p.processTask(ctx, task)
	}
}

func (p *Poller) processTask(ctx context.Context, task pollTask) {
	p.mu.Lock()
	curSeq, active := p.inFlight[task.tripID]
	if !active || curSeq != task.seq {
		p.mu.Unlock()
		p.recordPositionRequest(ctx, resultCancelled)
		return
	}
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		if p.inFlight[task.tripID] == task.seq {
			delete(p.inFlight, task.tripID)
		}
		p.mu.Unlock()
	}()

	opts := []trace.SpanStartOption{
		trace.WithNewRoot(),
		trace.WithAttributes(
			attribute.String("trip.id", task.tripID.String()),
			attribute.String("driver.id", task.driverID.String()),
		),
	}
	if task.scanSpanCtx.IsValid() {
		opts = append(opts, trace.WithLinks(trace.Link{SpanContext: task.scanSpanCtx}))
	}

	taskCtx, span := p.tracer.Start(ctx, "positions.process", opts...)
	defer span.End()

	p.recordWorkersActive(taskCtx, 1)
	defer p.recordWorkersActive(taskCtx, -1)

	reqCtx, cancel := context.WithTimeout(taskCtx, p.cfg.RequestTimeout)
	defer cancel()

	start := time.Now()
	err := p.pushGW.RequestPosition(reqCtx, task.driverID, task.tripID)
	duration := time.Since(start).Seconds()

	p.recordRequestDuration(taskCtx, duration)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "push_error"))
		p.recordPositionRequest(taskCtx, resultError)
		p.logger.ErrorContext(taskCtx, "push position request failed",
			slog.String("trip_id", task.tripID.String()),
			slog.String("driver_id", task.driverID.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	p.recordPositionRequest(taskCtx, resultOK)
	p.logger.InfoContext(taskCtx, "push position request sent",
		slog.String("trip_id", task.tripID.String()),
		slog.String("driver_id", task.driverID.String()),
	)
}

func (p *Poller) recordStaleCheck(ctx context.Context, result string) {
	if p.metrics.StaleChecks != nil {
		p.metrics.StaleChecks.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	}
}

func (p *Poller) recordQueueSize(ctx context.Context, delta int64) {
	if p.metrics.QueueSize != nil {
		p.metrics.QueueSize.Add(ctx, delta)
	}
}

func (p *Poller) recordWorkersActive(ctx context.Context, delta int64) {
	if p.metrics.WorkersActive != nil {
		p.metrics.WorkersActive.Add(ctx, delta)
	}
}

func (p *Poller) recordPositionRequest(ctx context.Context, result string) {
	if p.metrics.Requests != nil {
		p.metrics.Requests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
	}
}

func (p *Poller) recordRequestDuration(ctx context.Context, seconds float64) {
	if p.metrics.RequestDuration != nil {
		p.metrics.RequestDuration.Record(ctx, seconds)
	}
}
