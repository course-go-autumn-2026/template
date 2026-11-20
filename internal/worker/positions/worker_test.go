package positions

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

type fakeGateway struct{ calls chan uuid.UUID }

func (g fakeGateway) Request(_ context.Context, id, _ uuid.UUID) error { g.calls <- id; return nil }

type fakeTicker struct{ ch chan time.Time }

func (t fakeTicker) Chan() <-chan time.Time { return t.ch }
func (fakeTicker) Stop()                    {}
func makePool(t *testing.T, n int, g fakeGateway) *Pool {
	t.Helper()
	p, err := New(nil, g, slog.New(slog.NewTextHandler(io.Discard, nil)), noop.NewMeterProvider().Meter("test"), tracenoop.NewTracerProvider().Tracer("test"), time.Second, time.Minute, time.Second, time.Second, 100, n, 1)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPriorityDuplicateCancelAndDrain(t *testing.T) {
	ctx := context.Background()
	g := fakeGateway{make(chan uuid.UUID, 4)}
	p := makePool(t, 2, g)
	now := time.Now()
	a, b, c := Task{ID: uuid.New(), At: now}, Task{ID: uuid.New(), At: now.Add(-time.Minute)}, Task{ID: uuid.New(), At: now.Add(-2 * time.Minute)}
	if !p.enqueue(ctx, a) || !p.enqueue(ctx, b) || !p.enqueue(ctx, b) {
		t.Fatal("enqueue")
	}
	if p.enqueue(ctx, c) {
		t.Fatal("expected displacement")
	}
	p.Cancel(ctx, b.ID)
	p.start(ctx, fakeTicker{make(chan time.Time)})
	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case id := <-g.calls:
		if id != c.ID {
			t.Fatalf("got %s", id)
		}
	default:
		t.Fatal("missing task")
	}
	select {
	case <-g.calls:
		t.Fatal("unexpected task")
	default:
	}
}

func TestQueuePriority(t *testing.T) {
	ctx := context.Background()
	g := fakeGateway{make(chan uuid.UUID, 4)}
	p := makePool(t, 3, g)
	now := time.Now()
	a, b, c := Task{ID: uuid.New(), At: now}, Task{ID: uuid.New(), At: now.Add(-time.Minute)}, Task{ID: uuid.New(), At: now.Add(-2 * time.Minute)}
	p.enqueue(ctx, a)
	p.enqueue(ctx, c)
	p.enqueue(ctx, b)
	p.start(ctx, fakeTicker{make(chan time.Time)})
	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	for _, want := range []uuid.UUID{c.ID, b.ID, a.ID} {
		if got := <-g.calls; got != want {
			t.Fatalf("got %s want %s", got, want)
		}
	}
}

type failingGateway struct{ calls chan uuid.UUID }

func (g failingGateway) Request(_ context.Context, id, _ uuid.UUID) error {
	g.calls <- id
	return fmt.Errorf("push failed")
}

func TestWorkerContinuesAfterError(t *testing.T) {
	ctx := context.Background()
	calls := make(chan uuid.UUID, 2)
	p := makePool(t, 2, fakeGateway{calls})
	p.gateway = failingGateway{calls: calls}
	p.enqueue(ctx, Task{ID: uuid.New(), At: time.Now()})
	p.enqueue(ctx, Task{ID: uuid.New(), At: time.Now()})
	p.start(ctx, fakeTicker{make(chan time.Time)})
	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 {
		t.Fatalf("processed %d tasks", len(calls))
	}
}

func TestFullQueueDoesNotBlock(t *testing.T) {
	p := makePool(t, 1, fakeGateway{make(chan uuid.UUID, 1)})
	ctx := context.Background()
	now := time.Now()
	p.enqueue(ctx, Task{ID: uuid.New(), At: now})
	if p.enqueue(ctx, Task{ID: uuid.New(), At: now.Add(time.Second)}) {
		t.Fatal("expected dropped task")
	}
	if len(p.queue) != 1 {
		t.Fatal("queue capacity exceeded")
	}
}

type blockingGateway struct {
	started chan uuid.UUID
	release chan struct{}
}

func (g blockingGateway) Request(ctx context.Context, id, _ uuid.UUID) error {
	g.started <- id
	select {
	case <-g.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestFixedWorkerConcurrency(t *testing.T) {
	ctx := context.Background()
	p := makePool(t, 10, fakeGateway{make(chan uuid.UUID, 1)})
	g := blockingGateway{make(chan uuid.UUID, 10), make(chan struct{})}
	p.gateway = g
	p.workers = 3
	for i := 0; i < 8; i++ {
		p.enqueue(ctx, Task{ID: uuid.New(), At: time.Now()})
	}
	p.start(ctx, fakeTicker{make(chan time.Time)})
	for i := 0; i < 3; i++ {
		select {
		case <-g.started:
		case <-time.After(5 * time.Second):
			t.Fatal("workers did not start")
		}
	}
	select {
	case <-g.started:
		t.Fatal("more requests than workers")
	default:
	}
	close(g.release)
	if err := p.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if len(g.started) != 5 {
		t.Fatalf("remaining tasks: %d", len(g.started))
	}
}

func TestContextCancellationStopsPool(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := makePool(t, 1, fakeGateway{make(chan uuid.UUID, 1)})
	p.start(ctx, fakeTicker{make(chan time.Time)})
	cancel()
	done := make(chan struct{})
	go func() { p.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("workers did not stop after cancellation")
	}
	<-p.scanDone
}
