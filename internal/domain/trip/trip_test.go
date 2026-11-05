package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
)

func TestNewTrip(t *testing.T) {
	t.Parallel()

	validUser := uuid.New()
	validDriver := uuid.New()
	validPoint := trip.Point{Latitude: 59.93, Longitude: 30.31}
	now := time.Now().UTC()

	tests := []struct {
		name       string
		userID     uuid.UUID
		driverID   uuid.UUID
		startPoint trip.Point
		endPoint   trip.Point
		price      int64
		wantErr    bool
	}{
		{
			name:       "success",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validPoint,
			endPoint:   validPoint,
			price:      1450,
			wantErr:    false,
		},
		{
			name:       "empty user_id",
			userID:     uuid.Nil,
			driverID:   validDriver,
			startPoint: validPoint,
			endPoint:   validPoint,
			price:      1450,
			wantErr:    true,
		},
		{
			name:       "empty driver_id",
			userID:     validUser,
			driverID:   uuid.Nil,
			startPoint: validPoint,
			endPoint:   validPoint,
			price:      1450,
			wantErr:    true,
		},
		{
			name:       "invalid latitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: trip.Point{Latitude: 95.0, Longitude: 30.0},
			endPoint:   validPoint,
			price:      1450,
			wantErr:    true,
		},
		{
			name:       "invalid longitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validPoint,
			endPoint:   trip.Point{Latitude: 50.0, Longitude: 195.0},
			price:      1450,
			wantErr:    true,
		},
		{
			name:       "negative price",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validPoint,
			endPoint:   validPoint,
			price:      -100,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tr, err := trip.NewTrip(tt.userID, tt.driverID, tt.startPoint, tt.endPoint, tt.price, now)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, tr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tr)
				require.Equal(t, trip.StatusActive, tr.Status)
			}
		})
	}
}

func TestTrip_Finish(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	tr, err := trip.NewTrip(
		uuid.New(), uuid.New(),
		trip.Point{Latitude: 10, Longitude: 10},
		trip.Point{Latitude: 20, Longitude: 20},
		500, now,
	)
	require.NoError(t, err)

	finishAt := now.Add(15 * time.Minute)
	require.NoError(t, tr.Finish(finishAt))
	require.Equal(t, trip.StatusCompleted, tr.Status)
	require.Equal(t, &finishAt, tr.FinishedAt)

	err = tr.Finish(finishAt.Add(time.Minute))
	require.ErrorIs(t, err, trip.ErrTripCompleted)
}
