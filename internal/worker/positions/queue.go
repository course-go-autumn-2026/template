package positions

import (
	"container/heap"
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

type Task struct {
	Trip        trip.Trip
	SpanContext trace.SpanContext
	seq         int64
}

func effectiveTime(t trip.Trip) time.Time {
	if t.LastPositionAt != nil {
		return *t.LastPositionAt
	}
	return t.StartedAt
}

// taskHeap implements heap.Interface for min-heap ordered by effectiveTime.
// Trips that have been silent the longest (earliest effectiveTime) are at the top.
type taskHeap []*Task

func (h taskHeap) Len() int { return len(h) }

func (h taskHeap) Less(i, j int) bool {
	ti := effectiveTime(h[i].Trip)
	tj := effectiveTime(h[j].Trip)
	if ti.Equal(tj) {
		return h[i].seq < h[j].seq
	}
	return ti.Before(tj)
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *taskHeap) Push(x any) {
	item := x.(*Task)
	*h = append(*h, item)
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return item
}

type Queue struct {
	trips      chan Task
	mu         *sync.Mutex
	status     map[uuid.UUID]struct{}
	heap       taskHeap
	items      chan struct{}
	closedChan chan struct{}
	capacity   int
	seqCounter int64
	closed     bool
	metrics    scannerMetrics
	logger     *slog.Logger
}

func NewQueue(buffSize int, mu *sync.Mutex) *Queue {
	return &Queue{
		trips:      make(chan Task, buffSize),
		mu:         mu,
		status:     make(map[uuid.UUID]struct{}),
		heap:       make(taskHeap, 0, buffSize),
		items:      make(chan struct{}, buffSize),
		closedChan: make(chan struct{}),
		capacity:   buffSize,
	}
}

func (q *Queue) SetObservability(metrics scannerMetrics, logger *slog.Logger) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.metrics = metrics
	q.logger = logger
}

func (q *Queue) Enqueue(t trip.Trip, spanCtx trace.SpanContext, result *string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 1. Дедупликация: поездка уже в очереди
	if _, exists := q.status[t.ID]; exists {
		*result = staleCheckResultDuplicate
		return
	}

	q.seqCounter++

	// 2. Если очередь не заполнена: добавляем в кучу
	if len(q.heap) < q.capacity {
		task := &Task{Trip: t, SpanContext: spanCtx, seq: q.seqCounter}
		heap.Push(&q.heap, task)
		q.status[t.ID] = struct{}{}
		q.items <- struct{}{}
		*result = staleCheckResultQueued
		return
	}

	// 3. Очередь заполнена: ищем самую свежую поездку в очереди
	maxIdx := 0
	maxTime := effectiveTime(q.heap[0].Trip)
	for i := 1; i < len(q.heap); i++ {
		curTime := effectiveTime(q.heap[i].Trip)
		if curTime.After(maxTime) {
			maxTime = curTime
			maxIdx = i
		}
	}

	newTripTime := effectiveTime(t)
	// Если новая поездка молчит дольше (её время раньше maxTime) -> вытесняем самую свежую
	if newTripTime.Before(maxTime) {
		evicted := heap.Remove(&q.heap, maxIdx).(*Task)
		delete(q.status, evicted.Trip.ID)

		if q.metrics != nil {
			q.metrics.StaleCheck(context.Background(), staleCheckResultDropped)
		}
		if q.logger != nil {
			q.logger.Warn("position request queue is full, trip evicted", slog.String("trip_id", evicted.Trip.ID.String()))
		}

		task := &Task{Trip: t, SpanContext: spanCtx, seq: q.seqCounter}
		heap.Push(&q.heap, task)
		q.status[t.ID] = struct{}{}
		*result = staleCheckResultQueued
		return
	}

	// Новая поездка свежее всех -> дропается она сама
	*result = staleCheckResultDropped
}

func (q *Queue) Dequeue(ctx context.Context) (*Task, bool) {
	select {
	case <-ctx.Done():
		return nil, true
	case <-q.closedChan:
		q.mu.Lock()
		defer q.mu.Unlock()
		if len(q.heap) > 0 {
			task := heap.Pop(&q.heap).(*Task)
			select {
			case <-q.items:
			default:
			}
			return task, false
		}
		return nil, true
	case <-q.items:
		q.mu.Lock()
		defer q.mu.Unlock()
		if len(q.heap) == 0 {
			if q.closed {
				return nil, true
			}
			return nil, false
		}
		task := heap.Pop(&q.heap).(*Task)
		return task, false
	}
}

func (q *Queue) Delete(id uuid.UUID) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.status, id)
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.closedChan)
		close(q.trips)
	}
}
