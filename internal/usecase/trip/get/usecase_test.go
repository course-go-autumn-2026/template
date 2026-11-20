package get_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	"github.com/George1487/MalyginGeorge/internal/usecase/trip/get"
)

func TestUsecase_Execute(t *testing.T) {
	t.Parallel()
	type mocks struct {
		tripRepo *MocktripRepository
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
	expectedTrip := trip.Trip{
		ID:     tripID,
		Status: trip.StatusActive,
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
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(expectedTrip, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.NoError(t, err)
					require.Equal(t, expectedTrip, res)
				},
			},
		},
		{
			name: "trip not found",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
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
			name: "repository internal error",
			in: in{
				ctx: context.Background(),
				id:  tripID,
				setup: func(m mocks) {
					m.tripRepo.EXPECT().
						GetTrip(gomock.Any(), tripID).
						Return(trip.Trip{}, errors.New("db connection failure"))
				},
			},
			out: out{
				assert: func(t *testing.T, res trip.Trip, err error) {
					require.Error(t, err)
					require.Contains(t, err.Error(), "db connection failure")
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
				tripRepo: NewMocktripRepository(ctrl),
			}
			tt.in.setup(m)

			uc := get.New(m.tripRepo)
			res, err := uc.Execute(tt.in.ctx, tt.in.id)

			tt.out.assert(t, res, err)
		})
	}
}
