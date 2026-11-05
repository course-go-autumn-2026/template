package domainerror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "validation error -> 400",
			err:        trip.NewValidationError("price", "negative"),
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "not found -> 404",
			err:        trip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "completed -> 409",
			err:        trip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "driver busy -> 409",
			err:        trip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "internal db error -> 500 without leaking details",
			err:        errors.New("pq: password authentication failed"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, prob := domainerror.Problem(tt.err, "/api/v1/trips")
			require.Equal(t, tt.wantStatus, status)
			require.Equal(t, tt.wantCode, prob.Code)
			if status == http.StatusInternalServerError {
				require.Equal(t, "Internal server error", prob.Detail)
			}
		})
	}
}
