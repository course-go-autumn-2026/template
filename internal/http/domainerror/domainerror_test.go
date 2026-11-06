package domainerror_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/toooriii/tori/internal/domain/trip"
	"github.com/toooriii/tori/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "validation", err: &trip.ValidationError{Field: "price", Message: "must be >= 0"}, wantStatus: 400, wantCode: "invalid_request"},
		{name: "not found", err: trip.ErrTripNotFound, wantStatus: 404, wantCode: "trip_not_found"},
		{name: "completed", err: trip.ErrTripCompleted, wantStatus: 409, wantCode: "trip_completed"},
		{name: "driver busy", err: trip.ErrDriverBusy, wantStatus: 409, wantCode: "driver_busy"},
		{name: "idem conflict", err: trip.ErrIdempotencyConflict, wantStatus: 409, wantCode: "idempotency_conflict"},
		{name: "key exists", err: trip.ErrKeyAlreadyExists, wantStatus: 409, wantCode: "idempotency_conflict"},
		{name: "internal hides details", err: errors.New("db exploded: SELECT * FROM users"), wantStatus: 500, wantCode: "internal_error"},
		{name: "wrapped not found", err: errors.Join(trip.ErrTripNotFound, errors.New("ctx")), wantStatus: 404, wantCode: "trip_not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("code = %q, want %q", body.Code, tt.wantCode)
			}
			if tt.wantStatus == 500 && strings.Contains(deref(body.Detail), "db exploded") {
				t.Fatalf("internal error details leaked: %q", deref(body.Detail))
			}
		})
	}
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
