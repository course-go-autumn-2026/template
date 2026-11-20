package strategy_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror/strategy"
)

const unmatchedErrorCase = "unmatched error"

func TestValidationStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
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
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusBadRequest, status)
					require.Equal(t, "invalid_request", p.Code)
					require.Equal(t, "price: must be positive", *p.Detail)
				},
			},
		},
		{
			name: "invalid request sentinel error",
			in: in{
				err: trip.ErrInvalidRequest,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusBadRequest, status)
					require.Equal(t, "invalid_request", p.Code)
					require.Equal(t, "Request validation failed", *p.Detail)
				},
			},
		},
		{
			name: unmatchedErrorCase,
			in: in{
				err: errors.New("other error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.False(t, ok)
				},
			},
		},
	}

	strat := strategy.NewValidationStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}

func TestNotFoundStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "matched trip not found",
			in: in{
				err: trip.ErrTripNotFound,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusNotFound, status)
					require.Equal(t, "trip_not_found", p.Code)
				},
			},
		},
		{
			name: unmatchedErrorCase,
			in: in{
				err: errors.New("other error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.False(t, ok)
				},
			},
		},
	}

	strat := strategy.NewNotFoundStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}

func TestCompletedStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "matched trip completed",
			in: in{
				err: trip.ErrTripCompleted,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, "trip_completed", p.Code)
				},
			},
		},
		{
			name: unmatchedErrorCase,
			in: in{
				err: errors.New("other error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.False(t, ok)
				},
			},
		},
	}

	strat := strategy.NewCompletedStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}

func TestDriverBusyStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "matched driver busy",
			in: in{
				err: trip.ErrDriverBusy,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, "driver_busy", p.Code)
				},
			},
		},
		{
			name: unmatchedErrorCase,
			in: in{
				err: errors.New("other error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.False(t, ok)
				},
			},
		},
	}

	strat := strategy.NewDriverBusyStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}

func TestIdempotencyStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "matched idempotency conflict",
			in: in{
				err: trip.ErrIdempotencyConflict,
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusConflict, status)
					require.Equal(t, "idempotency_conflict", p.Code)
				},
			},
		},
		{
			name: unmatchedErrorCase,
			in: in{
				err: errors.New("other error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.False(t, ok)
				},
			},
		},
	}

	strat := strategy.NewIdempotencyStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}

func TestInternalStrategy_Handle(t *testing.T) {
	t.Parallel()
	type in struct {
		err error
	}

	type out struct {
		assert func(t *testing.T, status int, p api.Problem, ok bool)
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "fallback internal error",
			in: in{
				err: errors.New("any unexpected error"),
			},
			out: out{
				assert: func(t *testing.T, status int, p api.Problem, ok bool) {
					require.True(t, ok)
					require.Equal(t, http.StatusInternalServerError, status)
					require.Equal(t, "internal_error", p.Code)
				},
			},
		},
	}

	strat := strategy.NewInternalStrategy()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			status, p, ok := strat.Handle(tt.in.err)
			tt.out.assert(t, status, p, ok)
		})
	}
}
