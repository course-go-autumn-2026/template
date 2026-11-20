package domainerror_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
)

func TestProblem(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "validation error with field",
			in: in{
				err: &trip.ValidationError{Field: "price", Message: "must be positive"},
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusBadRequest, status)
					require.Equal(t, int32(400), p.Status)
					require.Equal(t, "invalid_request", p.Code)
					require.Equal(t, "price: must be positive", *p.Detail)
				},
			},
		},
		{
			name: "invalid request error",
			in: in{
				err: trip.ErrInvalidRequest,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusBadRequest, status)
					require.Equal(t, int32(400), p.Status)
					require.Equal(t, "invalid_request", p.Code)
					require.Equal(t, "Request validation failed", *p.Detail)
				},
			},
		},
		{
			name: "trip not found",
			in: in{
				err: trip.ErrTripNotFound,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusNotFound, status)
					require.Equal(t, int32(404), p.Status)
					require.Equal(t, "trip_not_found", p.Code)
				},
			},
		},
		{
			name: "trip completed",
			in: in{
				err: trip.ErrTripCompleted,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, int32(409), p.Status)
					require.Equal(t, "trip_completed", p.Code)
				},
			},
		},
		{
			name: "driver busy",
			in: in{
				err: trip.ErrDriverBusy,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, int32(409), p.Status)
					require.Equal(t, "driver_busy", p.Code)
				},
			},
		},
		{
			name: "idempotency conflict",
			in: in{
				err: trip.ErrIdempotencyConflict,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, int32(409), p.Status)
					require.Equal(t, "idempotency_conflict", p.Code)
				},
			},
		},
		{
			name: "unknown internal error",
			in: in{
				err: errors.New("db disconnected"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem) {
					require.Equal(t, http.StatusInternalServerError, status)
					require.Equal(t, int32(500), p.Status)
					require.Equal(t, "internal_error", p.Code)
					require.Equal(t, "Internal server error", *p.Detail)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p := domainerror.Problem(tt.in.err)
			tt.out.assert(t, status, p)
		})
	}
}

func TestWrite(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "write problem to response recorder",
			in: in{
				err: trip.ErrTripNotFound,
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusNotFound, rec.Code)
					require.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

					var p api.Problem
					err := json.NewDecoder(rec.Body).Decode(&p)
					require.NoError(t, err)
					require.Equal(t, "trip_not_found", p.Code)
					require.Equal(t, int32(404), p.Status)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			domainerror.Write(rec, tt.in.err)
			tt.out.assert(t, rec)
		})
	}
}
