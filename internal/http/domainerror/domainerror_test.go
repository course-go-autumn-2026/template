package domainerror_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	"github.com/n1ckerr0r/avito-labs/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"not found", trip.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"wrapped not found", fmt.Errorf("lookup trip: %w", trip.ErrTripNotFound), http.StatusNotFound, "trip_not_found"},
		{"completed", trip.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"driver busy", trip.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"conflict", trip.ErrConflict, http.StatusConflict, "conflict"},
		{"idempotency conflict", trip.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
		{"validation error", &trip.ValidationError{Field: "price", Reason: "must not be negative"}, http.StatusBadRequest, "invalid_request"},
		{"unknown error has no leaked details", errors.New("pq: connection refused"), http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, problem := domainerror.Problem(tt.err)

			assert.Equal(t, tt.wantStatus, status)
			assert.Equal(t, tt.wantCode, problem.Code)

			if tt.wantCode == "internal_error" {
				assert.NotContains(t, *problem.Detail, "pq:")
			}
		})
	}
}
