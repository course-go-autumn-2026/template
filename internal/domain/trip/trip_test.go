package trip_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eternal-despair/template/internal/domain/trip"
)

func TestTrip_Finish(t *testing.T) {
	t.Parallel()

	finishTime := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		initialStatus trip.Status
		wantErr       error
		checkAfter    func(t *testing.T, tr *trip.Trip)
	}{
		{
			name:          "finish active trip succeeds",
			initialStatus: trip.StatusActive,
			wantErr:       nil,
			checkAfter: func(t *testing.T, tr *trip.Trip) {
				assert.Equal(t, trip.StatusCompleted, tr.Status)
				require.NotNil(t, tr.FinishedAt)
				assert.Equal(t, finishTime, *tr.FinishedAt)
				assert.Equal(t, finishTime, tr.UpdatedAt)
			},
		},
		{
			name:          "finish already completed trip returns ErrTripCompleted",
			initialStatus: trip.StatusCompleted,
			wantErr:       trip.ErrTripCompleted,
			checkAfter: func(t *testing.T, tr *trip.Trip) {
				assert.Equal(t, trip.StatusCompleted, tr.Status)
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &trip.Trip{
				ID:        uuid.New(),
				Status:    tt.initialStatus,
				UpdatedAt: finishTime.Add(-10 * time.Minute),
			}

			err := tr.Finish(finishTime)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}

			if tt.checkAfter != nil {
				tt.checkAfter(t, tr)
			}
		})
	}
}

func TestNewTrip(t *testing.T) {
	t.Parallel()

	validUser := uuid.New()
	validDriver := uuid.New()
	validStart := trip.Coordinates{Latitude: 55.75, Longitude: 37.61}
	validEnd := trip.Coordinates{Latitude: 55.76, Longitude: 37.62}
	now := time.Now().UTC()

	tests := []struct {
		name       string
		userID     uuid.UUID
		driverID   uuid.UUID
		startPoint trip.Coordinates
		endPoint   trip.Coordinates
		price      int64
		wantErr    bool
		errField   string
	}{
		{
			name:       "valid trip creation",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validStart,
			endPoint:   validEnd,
			price:      1000,
			wantErr:    false,
		},
		{
			name:       "empty user id",
			userID:     uuid.Nil,
			driverID:   validDriver,
			startPoint: validStart,
			endPoint:   validEnd,
			price:      1000,
			wantErr:    true,
			errField:   "user_id",
		},
		{
			name:       "empty driver id",
			userID:     validUser,
			driverID:   uuid.Nil,
			startPoint: validStart,
			endPoint:   validEnd,
			price:      1000,
			wantErr:    true,
			errField:   "driver_id",
		},
		{
			name:       "invalid start latitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: trip.Coordinates{Latitude: 95.0, Longitude: 37.61},
			endPoint:   validEnd,
			price:      1000,
			wantErr:    true,
			errField:   "start_point.latitude",
		},
		{
			name:       "invalid start longitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: trip.Coordinates{Latitude: 55.75, Longitude: 185.0},
			endPoint:   validEnd,
			price:      1000,
			wantErr:    true,
			errField:   "start_point.longitude",
		},
		{
			name:       "invalid end latitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validStart,
			endPoint:   trip.Coordinates{Latitude: -95.0, Longitude: 37.62},
			price:      1000,
			wantErr:    true,
			errField:   "end_point.latitude",
		},
		{
			name:       "invalid end longitude",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validStart,
			endPoint:   trip.Coordinates{Latitude: 55.76, Longitude: -190.0},
			price:      1000,
			wantErr:    true,
			errField:   "end_point.longitude",
		},
		{
			name:       "negative price",
			userID:     validUser,
			driverID:   validDriver,
			startPoint: validStart,
			endPoint:   validEnd,
			price:      -50,
			wantErr:    true,
			errField:   "price",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr, err := trip.NewTrip(tt.userID, tt.driverID, tt.startPoint, tt.endPoint, tt.price, now)
			if tt.wantErr {
				require.Error(t, err)
				var valErr *trip.ValidationError
				require.ErrorAs(t, err, &valErr)
				assert.Equal(t, tt.errField, valErr.Field)
				assert.Nil(t, tr)
			} else {
				require.NoError(t, err)
				require.NotNil(t, tr)
				assert.NotEqual(t, uuid.Nil, tr.ID)
				assert.Equal(t, trip.StatusActive, tr.Status)
				assert.Equal(t, tt.userID, tr.UserID)
				assert.Equal(t, tt.driverID, tr.DriverID)
				assert.Equal(t, tt.price, tr.Price)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	valErr := trip.NewValidationError("user_id", "cannot be empty")
	assert.Equal(t, "user_id: cannot be empty", valErr.Error())
}
