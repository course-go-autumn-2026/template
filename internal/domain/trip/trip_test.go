package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
)

func TestTrip_Finish(t *testing.T) {
	t.Parallel()
	type in struct {
		trip trip.Trip
		at   time.Time
	}

	type out struct {
		assert func(t *testing.T, tr trip.Trip, err error)
	}

	finishTime := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	alreadyFinishedTime := finishTime.Add(-10 * time.Minute)

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success finish active trip",
			in: in{
				trip: trip.Trip{
					ID:         uuid.New(),
					Status:     trip.StatusActive,
					StartedAt:  finishTime.Add(-30 * time.Minute),
					FinishedAt: nil,
				},
				at: finishTime,
			},
			out: out{
				assert: func(t *testing.T, tr trip.Trip, err error) {
					require.NoError(t, err)
					require.Equal(t, trip.StatusCompleted, tr.Status)
					require.NotNil(t, tr.FinishedAt)
					require.Equal(t, finishTime, *tr.FinishedAt)
				},
			},
		},
		{
			name: "error finish already completed trip",
			in: in{
				trip: trip.Trip{
					ID:         uuid.New(),
					Status:     trip.StatusCompleted,
					StartedAt:  finishTime.Add(-30 * time.Minute),
					FinishedAt: &alreadyFinishedTime,
				},
				at: finishTime,
			},
			out: out{
				assert: func(t *testing.T, tr trip.Trip, err error) {
					require.Error(t, err)
					require.ErrorIs(t, err, trip.ErrTripCompleted)
					require.Equal(t, trip.StatusCompleted, tr.Status)
					require.Equal(t, &alreadyFinishedTime, tr.FinishedAt)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			currentTrip := tt.in.trip
			err := currentTrip.Finish(tt.in.at)
			tt.out.assert(t, currentTrip, err)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()
	type in struct {
		err trip.ValidationError
	}

	type out struct {
		assert func(t *testing.T, got string)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "with field",
			in: in{
				err: trip.ValidationError{
					Field:   "price",
					Message: "must be positive",
				},
			},
			out: out{
				assert: func(t *testing.T, got string) {
					require.Equal(t, "price: must be positive", got)
				},
			},
		},
		{
			name: "without field",
			in: in{
				err: trip.ValidationError{
					Field:   "",
					Message: "general error",
				},
			},
			out: out{
				assert: func(t *testing.T, got string) {
					require.Equal(t, "general error", got)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.in.err.Error()
			tt.out.assert(t, got)
		})
	}
}
