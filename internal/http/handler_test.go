package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/whxtelyy/template/internal/domain"
	api "github.com/whxtelyy/template/internal/generated"
	"github.com/whxtelyy/template/internal/service"
)

type createRepository struct {
	service.TripRepository
	calls int
}

func (r *createRepository) Create(_ context.Context, trip domain.Trip) (domain.Trip, error) {
	r.calls++
	return trip, nil
}

func (*createRepository) AddStatusHistory(context.Context, uuid.UUID, *string, string, string) error {
	return nil
}

type directTransaction struct{}

func (directTransaction) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestCreateTripContentType(t *testing.T) {
	for _, tc := range []struct {
		name, contentType string
		status, calls     int
	}{
		{"json", "application/json", http.StatusCreated, 1},
		{"charset", "application/json; charset=utf-8", http.StatusCreated, 1},
		{"missing", "", http.StatusBadRequest, 0},
		{"text", "text/plain", http.StatusBadRequest, 0},
		{"malformed", "application/json; charset", http.StatusBadRequest, 0},
		{"invalid", "not a media type", http.StatusBadRequest, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &createRepository{}
			h := NewHandler(service.NewTripService(repo, directTransaction{}), nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Second)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(`{"user_id":"11111111-1111-4111-8111-111111111111","driver_id":"22222222-2222-4222-8222-222222222222","start_point":{"latitude":1,"longitude":2},"end_point":{"latitude":3,"longitude":4},"price":100}`))
			r.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			h.CreateTrip(w, r, api.CreateTripParams{})
			if w.Code != tc.status || repo.calls != tc.calls {
				t.Fatalf("status=%d calls=%d body=%s", w.Code, repo.calls, w.Body.String())
			}
			if tc.status == http.StatusBadRequest {
				var problem api.Problem
				if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
					t.Fatal(err)
				}
				if problem.Code != "invalid_request" || problem.Status != int32(tc.status) || w.Header().Get("Content-Type") != "application/problem+json" {
					t.Fatalf("invalid problem response: %+v", problem)
				}
			}
		})
	}
}

type pingFunc func(context.Context) error

func (f pingFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestReady(t *testing.T) {
	for _, failed := range []bool{false, true} {
		h := NewHandler(nil, pingFunc(func(ctx context.Context) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Fatal("readiness ping must have a deadline")
			}
			if failed {
				return errors.New("unavailable")
			}
			return nil
		}), nil, time.Second)
		w := httptest.NewRecorder()
		h.Ready(w, httptest.NewRequest(http.MethodGet, "/ready", nil))
		want := http.StatusOK
		if failed {
			want = http.StatusServiceUnavailable
		}
		if w.Code != want {
			t.Fatalf("status=%d, want %d", w.Code, want)
		}
	}
}
