package positions_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

func TestQueue_Enqueue(t *testing.T) {
	t.Parallel()

	type in struct {
		bufferSize int
		setup      func(q *positions.Queue)
		trip       trip.Trip
		spanCtx    trace.SpanContext
	}

	type out struct {
		assert func(t *testing.T, q *positions.Queue, result string)
	}

	tripID := uuid.New()
	testTrip := trip.Trip{
		ID:       tripID,
		DriverID: uuid.New(),
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "enqueue new trip success",
			in: in{
				bufferSize: 10,
				setup:      func(q *positions.Queue) {},
				trip:       testTrip,
				spanCtx:    trace.SpanContext{},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, result string) {
					require.Equal(t, "queued", result)
					task, closed := q.Dequeue(context.Background())
					require.False(t, closed)
					require.NotNil(t, task)
					require.Equal(t, tripID, task.Trip.ID)
				},
			},
		},
		{
			name: "duplicate trip rejected",
			in: in{
				bufferSize: 10,
				setup: func(q *positions.Queue) {
					var res string
					q.Enqueue(testTrip, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)
				},
				trip:    testTrip,
				spanCtx: trace.SpanContext{},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, result string) {
					require.Equal(t, "duplicate", result)
				},
			},
		},
		{
			name: "dropped when queue is full",
			in: in{
				bufferSize: 1,
				setup: func(q *positions.Queue) {
					otherTrip := trip.Trip{ID: uuid.New(), DriverID: uuid.New()}
					var res string
					q.Enqueue(otherTrip, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)
				},
				trip:    testTrip,
				spanCtx: trace.SpanContext{},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, result string) {
					require.Equal(t, "dropped", result)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mu := &sync.Mutex{}
			q := positions.NewQueue(tt.in.bufferSize, mu)
			tt.in.setup(q)

			var result string
			q.Enqueue(tt.in.trip, tt.in.spanCtx, &result)

			tt.out.assert(t, q, result)
		})
	}
}

func TestQueue_Delete(t *testing.T) {
	t.Parallel()

	type in struct {
		bufferSize int
		setup      func(q *positions.Queue, tripID uuid.UUID)
		deleteID   func(tripID uuid.UUID) uuid.UUID
	}

	type out struct {
		assert func(t *testing.T, q *positions.Queue, tripID uuid.UUID)
	}

	tripID := uuid.New()
	testTrip := trip.Trip{
		ID:       tripID,
		DriverID: uuid.New(),
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "delete existing trip allows re-enqueuing",
			in: in{
				bufferSize: 10,
				setup: func(q *positions.Queue, id uuid.UUID) {
					var res string
					q.Enqueue(testTrip, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)
				},
				deleteID: func(id uuid.UUID) uuid.UUID {
					return id
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, id uuid.UUID) {
					var res string
					q.Enqueue(testTrip, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)
				},
			},
		},
		{
			name: "delete non-existent trip does nothing",
			in: in{
				bufferSize: 10,
				setup:      func(q *positions.Queue, id uuid.UUID) {},
				deleteID: func(id uuid.UUID) uuid.UUID {
					return uuid.New()
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue, id uuid.UUID) {
					var res string
					q.Enqueue(testTrip, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mu := &sync.Mutex{}
			q := positions.NewQueue(tt.in.bufferSize, mu)
			tt.in.setup(q, tripID)

			q.Delete(tt.in.deleteID(tripID))

			tt.out.assert(t, q, tripID)
		})
	}
}

func TestQueue_Dequeue(t *testing.T) {
	t.Parallel()

	type in struct {
		bufferSize int
		setup      func(q *positions.Queue, ctx context.Context, cancel context.CancelFunc)
	}

	type out struct {
		assert func(t *testing.T, task *positions.Task, contextStatus bool)
	}

	testTrip := trip.Trip{
		ID:       uuid.New(),
		DriverID: uuid.New(),
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "dequeue task from queue",
			in: in{
				bufferSize: 10,
				setup: func(q *positions.Queue, ctx context.Context, cancel context.CancelFunc) {
					var res string
					q.Enqueue(testTrip, trace.SpanContext{}, &res)
				},
			},
			out: out{
				assert: func(t *testing.T, task *positions.Task, contextStatus bool) {
					require.False(t, contextStatus)
					require.NotNil(t, task)
					require.Equal(t, testTrip.ID, task.Trip.ID)
				},
			},
		},
		{
			name: "dequeue on cancelled context returns nil and true",
			in: in{
				bufferSize: 10,
				setup: func(q *positions.Queue, ctx context.Context, cancel context.CancelFunc) {
					cancel()
				},
			},
			out: out{
				assert: func(t *testing.T, task *positions.Task, contextStatus bool) {
					require.True(t, contextStatus)
					require.Nil(t, task)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mu := &sync.Mutex{}
			q := positions.NewQueue(tt.in.bufferSize, mu)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tt.in.setup(q, ctx, cancel)

			task, contextStatus := q.Dequeue(ctx)

			tt.out.assert(t, task, contextStatus)
		})
	}
}

func TestQueue_PriorityOrder(t *testing.T) {
	t.Parallel()

	type in struct {
		trips []trip.Trip
	}

	type out struct {
		assert func(t *testing.T, dequeuedIDs, expectedIDs []uuid.UUID)
	}

	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	time2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	trip1 := trip.Trip{ID: uuid.New(), StartedAt: time1}
	trip2 := trip.Trip{ID: uuid.New(), StartedAt: time2}
	trip3 := trip.Trip{ID: uuid.New(), StartedAt: time3}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "trips enqueued in mixed order are dequeued starting with the one silent longest",
			in: in{
				trips: []trip.Trip{trip3, trip1, trip2}, // enqueued mixed: 12:00, 10:00, 11:00
			},
			out: out{
				assert: func(t *testing.T, dequeuedIDs, expectedIDs []uuid.UUID) {
					require.Equal(t, expectedIDs, dequeuedIDs)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mu := &sync.Mutex{}
			q := positions.NewQueue(10, mu)

			for _, tr := range tt.in.trips {
				var res string
				q.Enqueue(tr, trace.SpanContext{}, &res)
				require.Equal(t, "queued", res)
			}

			dequeuedIDs := make([]uuid.UUID, 0, len(tt.in.trips))
			for range tt.in.trips {
				task, closed := q.Dequeue(context.Background())
				require.False(t, closed)
				require.NotNil(t, task)
				dequeuedIDs = append(dequeuedIDs, task.Trip.ID)
			}

			expectedIDs := []uuid.UUID{trip1.ID, trip2.ID, trip3.ID}
			tt.out.assert(t, dequeuedIDs, expectedIDs)
		})
	}
}

func TestQueue_Eviction(t *testing.T) {
	t.Parallel()

	type in struct {
		capacity int
		action   func(q *positions.Queue)
	}

	type out struct {
		assert func(t *testing.T, q *positions.Queue)
	}

	time1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC) // oldest
	time2 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	time3 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) // freshest
	time4 := time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC) // even fresher

	trip1 := trip.Trip{ID: uuid.New(), StartedAt: time1}
	trip2 := trip.Trip{ID: uuid.New(), StartedAt: time2}
	trip3 := trip.Trip{ID: uuid.New(), StartedAt: time3}
	trip4 := trip.Trip{ID: uuid.New(), StartedAt: time4}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "older trip evicts freshest trip from full queue, freshest new trip is dropped itself",
			in: in{
				capacity: 2,
				action: func(q *positions.Queue) {
					var res string
					// Fill queue of 2 with trip1 (10:00) and trip3 (12:00)
					q.Enqueue(trip1, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)

					q.Enqueue(trip3, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)

					// Enqueue trip2 (11:00) - older than trip3 (12:00), so trip3 is evicted, trip2 queued
					q.Enqueue(trip2, trace.SpanContext{}, &res)
					require.Equal(t, "queued", res)

					// Enqueue trip4 (13:00) - fresher than everything in queue, so trip4 itself dropped
					q.Enqueue(trip4, trace.SpanContext{}, &res)
					require.Equal(t, "dropped", res)
				},
			},
			out: out{
				assert: func(t *testing.T, q *positions.Queue) {
					// Queue should contain trip1 (10:00) and trip2 (11:00)
					t1, _ := q.Dequeue(context.Background())
					require.Equal(t, trip1.ID, t1.Trip.ID)

					t2, _ := q.Dequeue(context.Background())
					require.Equal(t, trip2.ID, t2.Trip.ID)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mu := &sync.Mutex{}
			q := positions.NewQueue(tt.in.capacity, mu)

			tt.in.action(q)
			tt.out.assert(t, q)
		})
	}
}
