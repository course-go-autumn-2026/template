package positions_test

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
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"github.com/xotapy-main/courseGo/internal/model"
	"github.com/xotapy-main/courseGo/internal/worker/positions"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

type fakeTicker struct {
	ch chan time.Time
}

func newFakeTicker() *fakeTicker {
	return &fakeTicker{ch: make(chan time.Time)}
}

func (f *fakeTicker) C() <-chan time.Time {
	return f.ch
}

func (f *fakeTicker) Stop() {}

func (f *fakeTicker) Tick() {
	f.ch <- time.Now()
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestPoller_DeduplicatesAlreadyQueuedTrip(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	tripItem := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}

	startedFirst := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondScanDone := make(chan struct{})

	var scanCalls atomic.Int32
	repo.EXPECT().FindStaleActiveTrips(gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ time.Duration) ([]model.StaleTrip, error) {
			if scanCalls.Add(1) == 2 {
				defer close(secondScanDone)
			}
			return []model.StaleTrip{tripItem}, nil
		},
	).Times(2)

	var pushCalls atomic.Int32
	gw.EXPECT().RequestPosition(gomock.Any(), tripItem.DriverID, tripItem.ID).DoAndReturn(
		func(_ context.Context, _, _ uuid.UUID) error {
			pushCalls.Add(1)
			close(startedFirst)
			<-releaseFirst
			return nil
		},
	).Times(1)

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        1,
		QueueSize:      10,
		RequestTimeout: time.Second,
	})

	poller.Start(context.Background())

	ticker.Tick()
	<-startedFirst

	ticker.Tick()
	<-secondScanDone

	close(releaseFirst)
	require.NoError(t, poller.Shutdown(context.Background()))
	require.Equal(t, int32(1), pushCalls.Load())
}

func TestPoller_CancelBeforeExecutionSkipsPush(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	blockerTrip := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	cancelledTrip := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}

	blockerStarted := make(chan struct{})
	releaseBlocker := make(chan struct{})
	scanDone := make(chan struct{})

	repo.EXPECT().FindStaleActiveTrips(gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ time.Duration) ([]model.StaleTrip, error) {
			return []model.StaleTrip{blockerTrip, cancelledTrip}, nil
		},
	).Times(1)

	gw.EXPECT().RequestPosition(gomock.Any(), blockerTrip.DriverID, blockerTrip.ID).DoAndReturn(
		func(_ context.Context, _, _ uuid.UUID) error {
			close(blockerStarted)
			<-releaseBlocker
			return nil
		},
	).Times(1)

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        1,
		QueueSize:      10,
		RequestTimeout: time.Second,
	})

	poller.Start(context.Background())

	go func() {
		ticker.Tick()
		close(scanDone)
	}()

	<-blockerStarted
	<-scanDone

	poller.Cancel(cancelledTrip.ID)
	close(releaseBlocker)

	require.NoError(t, poller.Shutdown(context.Background()))
}

func TestPoller_QueueOverflowDropsWithoutBlockingScanner(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	trips := make([]model.StaleTrip, 10)
	for i := range trips {
		trips[i] = model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	}

	releaseWorker := make(chan struct{})
	secondTickReached := make(chan struct{})

	var scanCount atomic.Int32
	repo.EXPECT().FindStaleActiveTrips(gomock.Any(), time.Minute).DoAndReturn(
		func(_ context.Context, _ time.Duration) ([]model.StaleTrip, error) {
			if scanCount.Add(1) == 1 {
				return trips, nil
			}
			close(secondTickReached)
			return nil, nil
		},
	).Times(2)

	var processed atomic.Int32
	gw.EXPECT().RequestPosition(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ uuid.UUID) error {
			<-releaseWorker
			processed.Add(1)
			return nil
		},
	).AnyTimes()

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        1,
		QueueSize:      2,
		RequestTimeout: time.Second,
	})

	poller.Start(context.Background())

	ticker.Tick()
	ticker.Tick()

	select {
	case <-secondTickReached:
	case <-time.After(2 * time.Second):
		t.Fatal("scanner was blocked by full queue")
	}

	close(releaseWorker)
	require.NoError(t, poller.Shutdown(context.Background()))
	require.LessOrEqual(t, processed.Load(), int32(3))
}

func TestPoller_ProcessesNTasksWithFixedWorkers(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	const (
		totalTasks  = 12
		workerCount = 3
	)

	trips := make([]model.StaleTrip, totalTasks)
	for i := range trips {
		trips[i] = model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	}

	repo.EXPECT().FindStaleActiveTrips(gomock.Any(), time.Minute).Return(trips, nil).Times(1)

	var (
		inFlight    atomic.Int32
		maxInFlight atomic.Int32
		processed   atomic.Int32
		wg          sync.WaitGroup
	)
	wg.Add(totalTasks)

	gw.EXPECT().RequestPosition(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _, _ uuid.UUID) error {
			cur := inFlight.Add(1)
			for {
				prev := maxInFlight.Load()
				if cur <= prev || maxInFlight.CompareAndSwap(prev, cur) {
					break
				}
			}
			inFlight.Add(-1)
			processed.Add(1)
			wg.Done()
			return nil
		},
	).Times(totalTasks)

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        workerCount,
		QueueSize:      totalTasks,
		RequestTimeout: time.Second,
	})

	poller.Start(context.Background())
	ticker.Tick()
	wg.Wait()

	require.NoError(t, poller.Shutdown(context.Background()))
	require.Equal(t, int32(totalTasks), processed.Load())
	require.LessOrEqual(t, maxInFlight.Load(), int32(workerCount))
}

func TestPoller_StopsCleanlyOnContextCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        4,
		QueueSize:      10,
		RequestTimeout: time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	poller.Start(ctx)
	cancel()

	require.NoError(t, poller.Shutdown(context.Background()))
}

func TestPoller_PushErrorDoesNotCrashWorker(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := NewMockstaleTripRepository(ctrl)
	gw := NewMockpushGateway(ctrl)
	ticker := newFakeTicker()

	trip1 := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}
	trip2 := model.StaleTrip{ID: uuid.New(), DriverID: uuid.New()}

	repo.EXPECT().FindStaleActiveTrips(gomock.Any(), time.Minute).Return([]model.StaleTrip{trip1, trip2}, nil).Times(1)

	done := make(chan struct{})
	gomock.InOrder(
		gw.EXPECT().RequestPosition(gomock.Any(), trip1.DriverID, trip1.ID).Return(errors.New("push failed")),
		gw.EXPECT().RequestPosition(gomock.Any(), trip2.DriverID, trip2.ID).DoAndReturn(
			func(_ context.Context, _, _ uuid.UUID) error {
				close(done)
				return nil
			},
		),
	)

	poller := positions.New(repo, gw, ticker, discardLogger(), positions.Metrics{}, positions.Config{
		CheckInterval:  time.Second,
		StaleAfter:     time.Minute,
		Workers:        1,
		QueueSize:      10,
		RequestTimeout: time.Second,
	})

	poller.Start(context.Background())
	ticker.Tick()
	<-done

	require.NoError(t, poller.Shutdown(context.Background()))
}
