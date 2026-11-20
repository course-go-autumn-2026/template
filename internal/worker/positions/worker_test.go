package positions_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

func TestWorkerPool_Start(t *testing.T) {
	t.Parallel()

	type mocks struct {
		gateway *MockpushGateway
		metrics *MockworkerMetrics
	}

	type in struct {
		workersCount int
		setup        func(m mocks, q *positions.Queue, trips []trip.Trip)
		runAction    func(q *positions.Queue, trips []trip.Trip)
	}

	type out struct {
		assert func(t *testing.T, q *positions.Queue)
	}

	trip1 := trip.Trip{ID: uuid.New(), DriverID: uuid.New()}
	trip2 := trip.Trip{ID: uuid.New(), DriverID: uuid.New()}
	trip3 := trip.Trip{ID: uuid.New(), DriverID: uuid.New()}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "workers process tasks concurrently and record metrics",
			in: in{
				workersCount: 2,
				setup: func(m mocks, q *positions.Queue, trips []trip.Trip) {
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(1)).
						Times(2)
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(-1)).
						Times(2)
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(-1)).
						Times(2)
					m.metrics.EXPECT().
						RecordPositionRequestDuration(gomock.Any(), gomock.Any()).
						Times(2)
					m.metrics.EXPECT().
						PositionRequest(gomock.Any(), "success").
						Times(2)

					m.gateway.EXPECT().
						RequestPosition(gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil).
						Times(2)
				},
				runAction: func(q *positions.Queue, trips []trip.Trip) {
					for _, tr := range trips[:2] {
						var res string
						q.Enqueue(tr, trace.SpanContext{}, &res)
					}
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue) {
				},
			},
		},
		{
			name: "cancelled task when coordinates arrive is dropped without calling push",
			in: in{
				workersCount: 1,
				setup: func(m mocks, q *positions.Queue, trips []trip.Trip) {
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(1)).
						Times(1)
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(-1)).
						Times(1)
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(-1)).
						Times(1)
					m.metrics.EXPECT().
						PositionRequest(gomock.Any(), "dropped").
						Times(1)
				},
				runAction: func(q *positions.Queue, trips []trip.Trip) {
					var res string
					q.Enqueue(trips[0], trace.SpanContext{}, &res)
					q.Delete(trips[0].ID)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue) {
				},
			},
		},
		{
			name: "push service error does not crash worker and next task is processed",
			in: in{
				workersCount: 1,
				setup: func(m mocks, q *positions.Queue, trips []trip.Trip) {
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(1)).
						Times(2)
					m.metrics.EXPECT().
						AddActiveWorker(gomock.Any(), int64(-1)).
						Times(2)
					m.metrics.EXPECT().
						AddPositionQueueSize(gomock.Any(), int64(-1)).
						Times(2)
					m.metrics.EXPECT().
						RecordPositionRequestDuration(gomock.Any(), gomock.Any()).
						Times(2)

					m.metrics.EXPECT().
						PositionRequest(gomock.Any(), "error").
						Times(1)
					m.metrics.EXPECT().
						PositionRequest(gomock.Any(), "success").
						Times(1)

					gomock.InOrder(
						m.gateway.EXPECT().
							RequestPosition(gomock.Any(), trips[0].DriverID, trips[0].ID).
							Return(errors.New("push service 500 error")),
						m.gateway.EXPECT().
							RequestPosition(gomock.Any(), trips[1].DriverID, trips[1].ID).
							Return(nil),
					)
				},
				runAction: func(q *positions.Queue, trips []trip.Trip) {
					for _, tr := range trips[:2] {
						var res string
						q.Enqueue(tr, trace.SpanContext{}, &res)
					}
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue) {
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
				gateway: NewMockpushGateway(ctrl),
				metrics: NewMockworkerMetrics(ctrl),
			}

			mu := &sync.Mutex{}
			q := positions.NewQueue(10, mu)
			trips := []trip.Trip{trip1, trip2, trip3}
			tt.in.setup(m, q, trips)

			logger := slog.Default()
			pool := positions.NewWorkerPool(m.gateway, q, mu, m.metrics, logger)

			ctx, cancel := context.WithCancel(context.Background())
			pool.Start(tt.in.workersCount, ctx, 1*time.Second)

			tt.in.runAction(q, trips)
			q.Close()
			pool.Wait()
			cancel()

			tt.out.assert(t, q)
		})
	}
}
