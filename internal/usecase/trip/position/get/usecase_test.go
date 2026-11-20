package get_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/usecase/trip/position/get"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()

	type mocks struct {
		tripRepo     *MocktripRepository
		positionRepo *MocktripPositionRepository
		txManager    *MocktxManager
	}

	type in struct {
		ctx   context.Context
		id    uuid.UUID
		setup func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, res []trip.Position, err error)
	}

	tripID := uuid.New()
	testTrip := trip.Trip{
		ID:     tripID,
		Status: trip.StatusActive,
	}

	expectedPositions := []trip.Position{
		{
			ID:     1,
			TripID: tripID,
			Coordinates: trip.Coordinates{
				Latitude:  59.93,
				Longitude: 30.33,
			},
			RecordedAt: time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:     2,
			TripID: tripID,
			Coordinates: trip.Coordinates{
				Latitude:  59.94,
				Longitude: 30.34,
			},
			RecordedAt: time.Date(2026, 8, 24, 12, 1, 0, 0, time.UTC),
		},
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success: returns positions",
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
						GetTrip(gomock.Any(), tripID).
						Return(testTrip, nil)
					m.positionRepo.EXPECT().
						GetPositions(gomock.Any(), tripID).
						Return(expectedPositions, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, res []trip.Position, err error) {
					require.NoError(t, err)
					require.Equal(t, expectedPositions, res)
				},
			},
		},
		{
			name: "error: trip not found",
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
						GetTrip(gomock.Any(), tripID).
						Return(trip.Trip{}, trip.ErrTripNotFound)
				},
			},
			out: out{
				assert: func(t *testing.T, res []trip.Position, err error) {
					require.Error(t, err)
					require.ErrorIs(t, err, trip.ErrTripNotFound)
					require.Nil(t, res)
				},
			},
		},
		{
			name: "error: get positions fails",
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
						GetTrip(gomock.Any(), tripID).
						Return(testTrip, nil)
					m.positionRepo.EXPECT().
						GetPositions(gomock.Any(), tripID).
						Return(nil, errors.New("query error"))
				},
			},
			out: out{
				assert: func(t *testing.T, res []trip.Position, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "query error")
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
			}

			tt.in.setup(m)

			uc := get.NewUsecase(m.positionRepo, m.txManager, m.tripRepo)
			res, err := uc.Execute(tt.in.ctx, tt.in.id)

			tt.out.assert(t, res, err)
		})
	}
}
