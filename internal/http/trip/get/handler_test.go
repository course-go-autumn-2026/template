package get_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	api "github.com/freyliee/template/internal/generated"
	"github.com/freyliee/template/internal/http/trip/get"
)

func TestHandler(t *testing.T) {
	cases := []struct {
		name    string
		failure error
		status  int
		code    string
	}{{"success", nil, 200, ""}, {"missing", trip.ErrTripNotFound, 404, "trip_not_found"}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			usecase := NewMockusecase(ctrl)
			id := uuid.New()
			at := time.Now().UTC()
			entity := trip.Trip{ID: id, Status: trip.StatusCompleted, StartedAt: at, FinishedAt: &at}
			usecase.EXPECT().Execute(gomock.Any(), id).Return(entity, tc.failure)

			router := chi.NewRouter()
			router.Method("GET", "/api/v1/trips/{tripId}", get.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/trips/"+id.String()+"", nil))

			if w.Code != tc.status {
				t.Fatalf("unexpected status: %d %s", w.Code, w.Body.String())
			}

			if tc.failure == nil {
				var body api.Trip

				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}

				if body.Id != id || body.Status != api.Completed || body.FinishedAt == nil {
					t.Fatalf("unexpected trip: %+v", body)
				}
			} else {
				var problem api.Problem

				if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
					t.Fatal(err)
				}

				if problem.Code != tc.code || strings.Contains(w.Body.String(), "secret") {
					t.Fatalf("unexpected problem: %s", w.Body.String())
				}
			}
		})
	}
}

func TestInvalidID(t *testing.T) {
	ctrl := gomock.NewController(t)
	usecase := NewMockusecase(ctrl)
	router := chi.NewRouter()
	router.Method("GET", "/api/v1/trips/{tripId}", get.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", "/api/v1/trips/invalid", nil))

	if w.Code != 400 || !strings.Contains(w.Body.String(), "invalid_request") {
		t.Fatalf("unexpected invalid ID response: %s", w.Body.String())
	}
}
