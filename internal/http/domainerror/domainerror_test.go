package domainerror_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{{trip.ErrTripNotFound, 404, "trip_not_found"}, {trip.ErrTripCompleted, 409, "trip_completed"}, {trip.ErrDriverBusy, 409, "driver_busy"}, {&trip.ValidationError{Field: "price", Message: "must be non-negative"}, 400, "invalid_request"}, {errors.New("secret SQL password"), 500, "internal_error"}}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			status, body := domainerror.Problem(fmt.Errorf("wrapped: %w", tc.err))
			if status != tc.status || int(body.Status) != status || body.Code != tc.code || body.Title == "" || body.Type == "" || body.Detail == nil {
				t.Fatalf("unexpected problem: %+v", body)
			}

			if strings.Contains(*body.Detail, "secret") {
				t.Fatal("internal details leaked")
			}
		})
	}
}
