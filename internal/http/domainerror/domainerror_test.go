package domainerror

import (
	"errors"
	"testing"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

func TestProblem(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", domaintrip.ErrTripNotFound, 404, "trip_not_found"},
		{"completed", domaintrip.ErrTripCompleted, 409, "trip_completed"},
		{"driver busy", domaintrip.ErrDriverBusy, 409, "driver_busy"},
		{"idempotency conflict", domaintrip.ErrIdempotencyConflict, 409, "idempotency_conflict"},
		{"validation error", domaintrip.ValidationError{Field: "price", Reason: "must be >= 0"}, 400, "invalid_request"},
		{"unknown error", errors.New("boom"), 500, "internal_error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, body := Problem(c.err)
			if status != c.status {
				t.Fatalf("status = %d, want %d", status, c.status)
			}
			if body.Code != c.code {
				t.Fatalf("code = %s, want %s", body.Code, c.code)
			}
			if c.status == 500 {
				if body.Detail == nil || *body.Detail != "Unable to process the request" {
					t.Fatalf("500 should not leak details")
				}
			}
		})
	}
}
