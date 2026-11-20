package finish_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/usecase/trip/finish"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()
	type mocks struct {
		tripRepo        *MocktripRepository
		tripHistoryRepo *MocktripHistoryRepository
		txManager       *MocktxManager
		metrics         *MockmetricsRecorder
	}

	type in struct {
		ctx   context.Context
		id    uuid.UUID
		setup func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, res trip.Trip, err error)
	}

	tripID := uuid.New()
	activeTrip := trip.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		Status:     trip.StatusActive,
		StartedAt:  time.Now().Add(-10 * time.Minute),
		FinishedAt: nil,
	}

	now := time.Now()
	completedTrip := trip.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		Status:     trip.StatusCompleted,
		StartedAt:  now.Add(-20 * time.Minute),
		FinishedAt: &now,
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTripForUpdate(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.tripRepo.EXPECT().
						Update(gomock.Any(), gomock.Any()).
						Return(nil)
					m.tripHistoryRepo.EXPECT().
						AddTransition(gomock.Any(), tripID, gomock.Any(), trip.StatusCompleted, nil).
						Return(nil)
					m.metrics.EXPECT().
						TripCompleted(gomock.Any())
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.NoError(t, err)
					require.Equal(t, tripID, res.ID)
					require.Equal(t, trip.StatusCompleted, res.Status)
					require.NotNil(t, res.FinishedAt)
				},
			},
		},
		{
			name: "trip not found",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTripForUpdate(gomock.Any(), tripID).
						Return(trip.Trip{}, trip.ErrTripNotFound)
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.ErrorIs(t, err, trip.ErrTripNotFound)
					require.Empty(t, res)
				},
			},
		},
		{
			name: "trip already completed",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTripForUpdate(gomock.Any(), tripID).
						Return(completedTrip, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.ErrorIs(t, err, trip.ErrTripCompleted)
					require.Empty(t, res)
				},
			},
		},
		{
			name: "tripRepo.Update error",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTripForUpdate(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.tripRepo.EXPECT().
						Update(gomock.Any(), gomock.Any()).
						Return(errors.New("db update error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "db update error")
					require.Empty(t, res)
				},
			},
		},
		{
			name: "tripHistoryRepo.AddTransition error",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						GetTripForUpdate(gomock.Any(), tripID).
						Return(activeTrip, nil)
					m.tripRepo.EXPECT().
						Update(gomock.Any(), gomock.Any()).
						Return(nil)
					m.tripHistoryRepo.EXPECT().
						AddTransition(gomock.Any(), tripID, gomock.Any(), trip.StatusCompleted, nil).
						Return(errors.New("db insert history error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "db insert history error")
					require.Empty(t, res)
				},
			},
		},
		{
			name: "txManager error",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						Return(errors.New("tx error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "tx error")
					require.Empty(t, res)
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
				tripRepo:        NewMocktripRepository(ctrl),
				tripHistoryRepo: NewMocktripHistoryRepository(ctrl),
				txManager:       NewMocktxManager(ctrl),
				metrics:         NewMockmetricsRecorder(ctrl),
			}
			tt.in.setup(m)

			uc := finish.New(m.tripRepo, m.tripHistoryRepo, m.txManager, m.metrics)
			res, err := uc.Execute(tt.in.ctx, tt.in.id)

			tt.out.assert(t, res, err)
		})
	}
}
