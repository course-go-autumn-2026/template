package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/model"
	"github.com/George1487/MalyginGeorge/internal/usecase/trip/create"
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
		data  model.CreateTripData
		setup func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, res trip.Trip, err error)
	}

	validData := model.CreateTripData{
		UserID:   uuid.New(),
		DriverID: uuid.New(),
		StartPoint: trip.Coordinates{
			Latitude:  55.7558,
			Longitude: 37.6173,
		},
		EndPoint: trip.Coordinates{
			Latitude:  55.7522,
			Longitude: 37.6155,
		},
		Price: 50000,
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						Save(gomock.Any(), gomock.Any()).
						Return(nil)
					m.tripHistoryRepo.EXPECT().
						AddTransition(gomock.Any(), gomock.Any(), nil, trip.StatusActive, nil).
						Return(nil)
					m.metrics.EXPECT().
						TripCreated(gomock.Any())
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.NoError(t, err)
					require.NotEqual(t, uuid.Nil, res.ID)
					require.Equal(t, validData.UserID, res.UserID)
					require.Equal(t, validData.DriverID, res.DriverID)
					require.Equal(t, validData.StartPoint, res.StartPoint)
					require.Equal(t, validData.EndPoint, res.EndPoint)
					require.Equal(t, validData.Price, res.Price)
					require.Equal(t, trip.StatusActive, res.Status)
					require.False(t, res.StartedAt.IsZero())
					require.Nil(t, res.FinishedAt)
				},
			},
		},
		{
			name: "txManager error",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						Return(errors.New("tx start error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "tx start error")
					require.Empty(t, res)
				},
			},
		},
		{
			name: "tripRepo.Save error",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						Save(gomock.Any(), gomock.Any()).
						Return(errors.New("db insert trip error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "db insert trip error")
					require.Empty(t, res)
				},
			},
		},
		{
			name: "tripHistoryRepo.AddTransition error",
			in: in{
				ctx:  context.Background(),
				data: validData,
				setup: func(m mocks) {
					m.txManager.EXPECT().
						Do(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
							return fn(ctx)
						})
					m.tripRepo.EXPECT().
						Save(gomock.Any(), gomock.Any()).
						Return(nil)
					m.tripHistoryRepo.EXPECT().
						AddTransition(gomock.Any(), gomock.Any(), nil, trip.StatusActive, nil).
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

			uc := create.New(m.tripRepo, m.tripHistoryRepo, m.txManager, m.metrics)
			res, err := uc.Execute(tt.in.ctx, tt.in.data)

			tt.out.assert(t, res, err)
		})
	}
}
