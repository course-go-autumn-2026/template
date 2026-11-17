package positions

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
)

const queuedResult = "queued"

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func wait[T any](t *testing.T, ch <-chan T) T {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for synchronization")

		var zero T

		return zero
	}
}

func fixture(t *testing.T, workers, size int) (*Pool, *Mockrepository, *Mockgateway, *Mockobserver, *clockwork.FakeClock) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := NewMockrepository(ctrl)
	gateway := NewMockgateway(ctrl)
	observer := NewMockobserver(ctrl)
	clock := clockwork.NewFakeClockAt(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC))
	pool := New(
		Config{
			Interval:       30 * time.Second,
			StaleAfter:     time.Minute,
			Workers:        workers,
			QueueSize:      size,
			RequestTimeout: time.Second,
			QueryTimeout:   time.Second,
		},
		repo,
		gateway,
		clock,
		observer,
		slog.New(slog.NewJSONHandler(io.Discard, nil)),
		noop.NewTracerProvider().Tracer("test"),
	)

	return pool, repo, gateway, observer, clock
}

func start(t *testing.T, p *Pool, clock *clockwork.FakeClock) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	p.Start(ctx)
	t.Cleanup(func() {
		cancel()
		ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()

		if err := p.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err := clock.BlockUntilContext(ctx, 1); err != nil {
		t.Fatal(err)
	}

	return cancel
}

func entity() trip.StaleTrip { return trip.StaleTrip{ID: uuid.New(), DriverID: uuid.New()} }

func TestScannerDeduplicationCancellationAndOverflow(t *testing.T) {
	p, repo, gateway, observer, clock := fixture(t, 1, 1)
	a, b, c := entity(), entity(), entity()
	entered, release := make(chan struct{}), make(chan struct{})

	var releaseOnce sync.Once

	defer releaseOnce.Do(func() { close(release) })

	repo.EXPECT().NeedsPosition(gomock.Any(), a.ID, time.Minute).Return(true, nil)
	gateway.EXPECT().RequestPosition(gomock.Any(), a.ID, a.DriverID).DoAndReturn(func(ctx context.Context, _, _ uuid.UUID) error {
		close(entered)

		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})
	checked := make(chan string, 10)
	observer.EXPECT().StaleCheck(gomock.Any(), gomock.Any()).Do(func(_ context.Context, result string) { checked <- result }).Times(4)
	observer.EXPECT().PositionRequest(gomock.Any(), "dropped", time.Duration(0))
	observer.EXPECT().PositionRequest(gomock.Any(), "success", gomock.Any())
	gomock.InOrder(
		repo.EXPECT().FindStaleTrips(gomock.Any(), time.Minute).Return([]trip.StaleTrip{a}, nil),
		repo.EXPECT().FindStaleTrips(gomock.Any(), time.Minute).Return([]trip.StaleTrip{b}, nil),
		repo.EXPECT().FindStaleTrips(gomock.Any(), time.Minute).Return([]trip.StaleTrip{b, c}, nil),
	)
	start(t, p, clock)
	clock.Advance(30 * time.Second)
	wait(t, entered)

	if wait(t, checked) != queuedResult {
		t.Fatal("first trip not queued")
	}

	clock.Advance(30 * time.Second)

	if wait(t, checked) != queuedResult {
		t.Fatal("second trip not queued")
	}

	clock.Advance(30 * time.Second)

	if wait(t, checked) != "duplicate" || wait(t, checked) != "dropped" {
		t.Fatal("queue did not deduplicate or drop")
	}

	queued, active := p.Snapshot()

	if queued != 1 || active != 1 {
		t.Fatalf("invalid pool size: %d/%d", queued, active)
	}

	p.Cancel(b.ID)
	releaseOnce.Do(func() { close(release) })
}

func TestPoolFixedConcurrencyDrainsAndSurvivesError(t *testing.T) {
	p, repo, gateway, observer, clock := fixture(t, 3, 20)
	entered := make(chan struct{}, 12)
	release := make(chan struct{})

	var once sync.Once

	defer once.Do(func() { close(release) })

	var active, maxActive, processed atomic.Int64

	repo.EXPECT().NeedsPosition(gomock.Any(), gomock.Any(), time.Minute).Return(true, nil).Times(12)
	gateway.EXPECT().RequestPosition(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, _, _ uuid.UUID) error {
		current := active.Add(1)
		defer active.Add(-1)

		for old := maxActive.Load(); current > old; old = maxActive.Load() {
			if maxActive.CompareAndSwap(old, current) {
				break
			}
		}

		entered <- struct{}{}

		select {
		case <-release:
		case <-ctx.Done():
			return ctx.Err()
		}

		if processed.Add(1) == 1 {
			return errors.New("push unavailable")
		}

		return nil
	}).Times(12)
	observer.EXPECT().PositionRequest(gomock.Any(), "error", gomock.Any()).Times(1)
	observer.EXPECT().PositionRequest(gomock.Any(), "success", gomock.Any()).Times(11)

	for range 12 {
		if result := p.enqueue(entity(), trace.SpanContext{}); result != queuedResult {
			t.Fatal(result)
		}
	}

	cancel := start(t, p, clock)

	for range 3 {
		wait(t, entered)
	}

	if maxActive.Load() != 3 {
		t.Fatal("worker count not respected")
	}

	cancel()
	once.Do(func() { close(release) })
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()

	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	if processed.Load() != 12 || maxActive.Load() != 3 {
		t.Fatalf("processed %d, concurrency %d", processed.Load(), maxActive.Load())
	}

	queued, busy := p.Snapshot()

	if queued != 0 || busy != 0 || len(p.pending) != 0 {
		t.Fatal("pool not drained")
	}
}

func TestCancellationTokensAndFreshness(t *testing.T) {
	p, repo, _, _, _ := fixture(t, 1, 3)
	entity := entity()

	if p.enqueue(entity, trace.SpanContext{}) != queuedResult {
		t.Fatal("enqueue")
	}

	p.Cancel(entity.ID)

	if p.enqueue(entity, trace.SpanContext{}) != queuedResult {
		t.Fatal("re-enqueue")
	}

	old, current := <-p.queue, <-p.queue
	p.process(context.Background(), old)

	if p.pending[entity.ID] != current {
		t.Fatal("old task removed replacement")
	}

	repo.EXPECT().NeedsPosition(gomock.Any(), entity.ID, time.Minute).Return(false, nil)
	p.process(context.Background(), current)

	if len(p.pending) != 0 {
		t.Fatal("fresh task not removed")
	}
}

func TestShutdownBudgetCancelsInFlight(t *testing.T) {
	p, repo, gateway, observer, clock := fixture(t, 1, 2)
	entity := entity()
	p.enqueue(entity, trace.SpanContext{})
	entered := make(chan struct{})
	repo.EXPECT().NeedsPosition(gomock.Any(), entity.ID, time.Minute).Return(true, nil)
	gateway.EXPECT().RequestPosition(gomock.Any(), entity.ID, entity.DriverID).DoAndReturn(func(ctx context.Context, _, _ uuid.UUID) error {
		close(entered)
		<-ctx.Done()

		return ctx.Err()
	})
	observer.EXPECT().PositionRequest(gomock.Any(), "error", gomock.Any())
	start(t, p, clock)
	wait(t, entered)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := p.Stop(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}
