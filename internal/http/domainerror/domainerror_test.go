package domainerror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	domain "github.com/whxtelyy/template/internal/domain/trip"
)

func TestProblem(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"missing", domain.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"completed", domain.ErrTripCompleted, http.StatusConflict, "trip_completed"},
		{"busy", domain.ErrDriverBusy, http.StatusConflict, "driver_busy"},
		{"idempotency", domain.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
		{"validation", &domain.ValidationError{Field: "price", Reason: "invalid"}, http.StatusBadRequest, "invalid_request"},
		{"internal", fmt.Errorf("database password: %w", errors.New("secret")), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := Problem(tc.err)
			if status != tc.status || body.Code != tc.code || int(body.Status) != status {
				t.Fatalf("Problem() = %d %+v", status, body)
			}
			if tc.name == "internal" && body.Detail != nil && *body.Detail != "Internal server error" {
				t.Fatalf("internal detail leaked: %s", *body.Detail)
			}
		})
	}
}
