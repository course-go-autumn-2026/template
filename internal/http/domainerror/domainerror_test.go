package domainerror_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eternal-despair/template/internal/domain/trip"
	"github.com/eternal-despair/template/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		expectNoLeak   bool
	}{
		{
			name:           "ErrDriverBusy -> 409 driver_busy",
			err:            trip.ErrDriverBusy,
			expectedStatus: http.StatusConflict,
			expectedCode:   "driver_busy",
		},
		{
			name:           "ErrTripCompleted -> 409 trip_completed",
			err:            trip.ErrTripCompleted,
			expectedStatus: http.StatusConflict,
			expectedCode:   "trip_completed",
		},
		{
			name:           "ErrTripNotFound -> 404 trip_not_found",
			err:            trip.ErrTripNotFound,
			expectedStatus: http.StatusNotFound,
			expectedCode:   "trip_not_found",
		},
		{
			name:           "ValidationError -> 400 invalid_request",
			err:            trip.NewValidationError("user_id", "cannot be empty"),
			expectedStatus: http.StatusBadRequest,
			expectedCode:   "invalid_request",
		},
		{
			name:           "Repository error -> 500 internal_error with NO internal details",
			err:            errors.New("FATAL: connection to postgres failed on port 5432: query SELECT * FROM trips"),
			expectedStatus: http.StatusInternalServerError,
			expectedCode:   "internal_error",
			expectNoLeak:   true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status, prob := domainerror.Problem(tt.err)
			assert.Equal(t, tt.expectedStatus, status)
			assert.Equal(t, tt.expectedCode, prob.Code)
			assert.Equal(t, int32(tt.expectedStatus), prob.Status) //nolint:gosec // HTTP status code is in valid range

			if tt.expectNoLeak {
				require.NotNil(t, prob.Detail)
				assert.NotContains(t, *prob.Detail, "postgres")
				assert.NotContains(t, *prob.Detail, "SELECT")
				assert.Equal(t, "Internal server error", *prob.Detail)
			}
		})
	}
}
