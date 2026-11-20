package positions_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

func TestScaner_Run(t *testing.T) {
	t.Parallel()

	type mocks struct {
		repo    *MockTripSurveyRepository
		metrics *MockscannerMetrics
	}

	type in struct {
		queueSize int
		batchSize int
		setup     func(m mocks, q *positions.Queue)
		advance   func(clock *clockwork.FakeClock)
	}

	type out struct {
		assert func(t *testing.T, q *positions.Queue, err error)
	}

	testTrip1 := trip.Trip{ID: uuid.New(), DriverID: uuid.New(), Status: trip.StatusActive, StartedAt: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)}
	testTrip2 := trip.Trip{ID: uuid.New(), DriverID: uuid.New(), Status: trip.StatusActive, StartedAt: time.Date(2026, 1, 1, 10, 1, 0, 0, time.UTC)}
	testTrip3 := trip.Trip{ID: uuid.New(), DriverID: uuid.New(), Status: trip.StatusActive, StartedAt: time.Date(2026, 1, 1, 10, 2, 0, 0, time.UTC)}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "stale trips successfully enqueued on tick",
			in: in{
				queueSize: 10,
				batchSize: 10,
				setup: func(m mocks, q *positions.Queue) {
					m.repo.EXPECT().
						GetTripsSurvey(gomock.Any(), 60*time.Second, 10, gomock.Any(), gomock.Any()).
						Return([]trip.Trip{testTrip1}, nil)

					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "queued")
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(1))
				},
				advance: func(clock *clockwork.FakeClock) {
					clock.Advance(30 * time.Second)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, err error) {
					require.ErrorIs(t, err, context.Canceled)
					task, closed := q.Dequeue(context.Background())
					require.False(t, closed)
					require.NotNil(t, task)
					require.Equal(t, testTrip1.ID, task.Trip.ID)
				},
			},
		},
		{
			name: "batching: multiple batches queried when queue has capacity",
			in: in{
				queueSize: 10,
				batchSize: 2,
				setup: func(m mocks, q *positions.Queue) {
					gomock.InOrder(
						m.repo.EXPECT().
							GetTripsSurvey(gomock.Any(), 60*time.Second, 2, time.Unix(0, 0), uuid.Nil).
							Return([]trip.Trip{testTrip1, testTrip2}, nil),
						m.repo.EXPECT().
							GetTripsSurvey(gomock.Any(), 60*time.Second, 2, testTrip2.StartedAt, testTrip2.ID).
							Return([]trip.Trip{testTrip3}, nil),
					)

					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "queued").
						Times(3)
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(1)).
						Times(3)
				},
				advance: func(clock *clockwork.FakeClock) {
					clock.Advance(30 * time.Second)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, err error) {
					require.ErrorIs(t, err, context.Canceled)
				},
			},
		},
		{
			name: "batching: stops after first batch if queue overflows (dropped)",
			in: in{
				queueSize: 1,
				batchSize: 2,
				setup: func(m mocks, q *positions.Queue) {
					m.repo.EXPECT().
						GetTripsSurvey(gomock.Any(), 60*time.Second, 2, time.Unix(0, 0), uuid.Nil).
						Return([]trip.Trip{testTrip1, testTrip2}, nil).
						Times(1)

					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "queued").
						Times(1)
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(1)).
						Times(1)

					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "dropped").
						Times(1)
				},
				advance: func(clock *clockwork.FakeClock) {
					clock.Advance(30 * time.Second)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, err error) {
					require.ErrorIs(t, err, context.Canceled)
				},
			},
		},
		{
			name: "duplicate trip on repeated tick gets duplicate metric",
			in: in{
				queueSize: 10,
				batchSize: 10,
				setup: func(m mocks, q *positions.Queue) {
					m.repo.EXPECT().
						GetTripsSurvey(gomock.Any(), 60*time.Second, 10, gomock.Any(), gomock.Any()).
						Return([]trip.Trip{testTrip1}, nil)
					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "queued")
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(1))

					m.repo.EXPECT().
						GetTripsSurvey(gomock.Any(), 60*time.Second, 10, gomock.Any(), gomock.Any()).
						Return([]trip.Trip{testTrip1}, nil)
					m.metrics.EXPECT().
						StaleCheck(gomock.Any(), "duplicate")
				},
				advance: func(clock *clockwork.FakeClock) {
					clock.Advance(30 * time.Second)
					time.Sleep(20 * time.Millisecond)
					clock.Advance(30 * time.Second)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, err error) {
					require.ErrorIs(t, err, context.Canceled)
					task, closed := q.Dequeue(context.Background())
					require.False(t, closed)
					require.NotNil(t, task)
					require.Equal(t, testTrip1.ID, task.Trip.ID)
				},
			},
		},
		{
			name: "repo error logged and scan continues on next tick",
			in: in{
				queueSize: 10,
				batchSize: 10,
				setup: func(m mocks, q *positions.Queue) {
					m.repo.EXPECT().
						GetTripsSurvey(gomock.Any(), 60*time.Second, 10, gomock.Any(), gomock.Any()).
						Return(nil, errors.New("db connection failed"))
				},
				advance: func(clock *clockwork.FakeClock) {
					clock.Advance(30 * time.Second)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, err error) {
					require.ErrorIs(t, err, context.Canceled)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mocks{
				repo:    NewMockTripSurveyRepository(ctrl),
				metrics: NewMockscannerMetrics(ctrl),
			}

			mu := &sync.Mutex{}
			q := positions.NewQueue(tt.in.queueSize, mu)
			tt.in.setup(m, q)

			fakeClock := clockwork.NewFakeClock()
			ticker := fakeClock.NewTicker(30 * time.Second)
			logger := slog.Default()

			scaner := positions.New(m.repo, ticker, m.metrics, logger, q)

			ctx, cancel := context.WithCancel(context.Background())
			errCh := make(chan error, 1)
			batchSize := tt.in.batchSize
			if batchSize == 0 {
				batchSize = 10
			}
			go func() {
				errCh <- scaner.Run(ctx, 30*time.Second, 60*time.Second, batchSize)
			}()

			require.NoError(t, fakeClock.BlockUntilContext(ctx, 1))
			tt.in.advance(fakeClock)

			time.Sleep(50 * time.Millisecond)
			cancel()
			err := <-errCh

			tt.out.assert(t, q, err)
		})
	}
}
