package create_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/model"
	"github.com/George1487/MalyginGeorge/internal/usecase/trip/position/create"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	type mocks struct {
		tripRepo     *MocktripRepository
		positionRepo *MocktripPositionRepository
		txManager    *MocktxManager
		metrics      *MockmetricsRecorder
	}

	type in struct {
		ctx   context.Context
		data  model.CreatePosition
		setup func(m mocks, q *positions.Queue)
	}

	type out struct {
		assert func(t *testing.T, res *trip.Position, q *positions.Queue, err error)
	}

	tripID := uuid.New()
	recordedAt := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	validData := model.CreatePosition{
		TripID:     tripID,
		Latitude:   59.9350,
		Longitude:  30.3350,
		RecordedAt: recordedAt,
	}

	activeTrip := trip.Trip{
		ID:     tripID,
		Status: trip.StatusActive,
	}

	completedTrip := trip.Trip{
		ID:     tripID,
		Status: trip.StatusCompleted,
	}

	expectedPosition := &trip.Position{
		ID:     1,
		TripID: tripID,
		Coordinates: trip.Coordinates{
			Latitude:  59.9350,
			Longitude: 30.3350,
		},
		RecordedAt: recordedAt,
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success: active trip saves position, updates last_position_at, removes from queue and records metric",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks, q *positions.Queue) {
					var res string
					q.Enqueue(activeTrip, trace.SpanContext{}, &res)

					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.positionRepo.EXPECT().
						AddPosition(gomock.Any(), gomock.Any()).
						Return(expectedPosition, nil)
					m.positionRepo.EXPECT().
						UpdateLastPositionAt(gomock.Any(), tripID, recordedAt).
						Return(nil)
					m.metrics.EXPECT().
						PositionSaved(gomock.Any())
				},
			},
			out: out{
				assert: func(t *testing.T, res *trip.Position, q *positions.Queue, err error) {
					require.NoError(t, err)
					require.Equal(t, expectedPosition, res)
					var resEnq string
					q.Enqueue(activeTrip, trace.SpanContext{}, &resEnq)
					require.Equal(t, "queued", resEnq)
				},
			},
		},
		{
			name: "error: completed trip returns ErrTripCompleted",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks, q *positions.Queue) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(completedTrip, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, res *trip.Position, q *positions.Queue, err error) {
					require.Error(t, err)
					require.ErrorIs(t, err, trip.ErrTripCompleted)
					require.Nil(t, res)
				},
			},
		},
		{
			name: "error: get trip repo failure",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks, q *positions.Queue) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(trip.Trip{}, errors.New("db error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res *trip.Position, q *positions.Queue, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "db error")
					require.Nil(t, res)
				},
			},
		},
		{
			name: "error: add position failure",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks, q *positions.Queue) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.positionRepo.EXPECT().
						AddPosition(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("insert error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res *trip.Position, q *positions.Queue, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "insert error")
					require.Nil(t, res)
				},
			},
		},
		{
			name: "error: update last position failure",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks, q *positions.Queue) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.positionRepo.EXPECT().
						AddPosition(gomock.Any(), gomock.Any()).
						Return(expectedPosition, nil)
					m.positionRepo.EXPECT().
						UpdateLastPositionAt(gomock.Any(), tripID, recordedAt).
						Return(errors.New("update error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res *trip.Position, q *positions.Queue, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "update error")
					require.Nil(t, res)
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
				tripRepo:     NewMocktripRepository(ctrl),
				positionRepo: NewMocktripPositionRepository(ctrl),
				txManager:    NewMocktxManager(ctrl),
				metrics:      NewMockmetricsRecorder(ctrl),
			}

			mu := &sync.Mutex{}
			q := positions.NewQueue(10, mu)
			tt.in.setup(m, q)

			uc := create.New(m.tripRepo, m.positionRepo, m.txManager, m.metrics, q)
			res, err := uc.Execute(tt.in.ctx, tt.in.data)

			tt.out.assert(t, res, q, err)
		})
	}
}
