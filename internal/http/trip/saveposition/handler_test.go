package saveposition_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	"github.com/freyliee/template/internal/http/trip/saveposition"
)

const validBody = `{"coordinates":{"latitude":0,"longitude":0},"recorded_at":"2026-11-15T00:00:00Z"}`

func TestHandler(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		err        error
		status     int
	}{
		{"success", validBody, nil, 201},
		{"missing trip", validBody, trip.ErrTripNotFound, 404},
		{"completed", validBody, trip.ErrTripCompleted, 409},
		{"storage", validBody, errors.New("secret database error"), 500},
		{
			"range",
			`{"coordinates":{"latitude":91,"longitude":0},"recorded_at":"2026-11-15T00:00:00Z"}`,
			&trip.ValidationError{Field: "coordinates.latitude", Message: "outside range"},
			400,
		},
		{
			"missing",
			`{"coordinates":{"latitude":0},"recorded_at":"2026-11-15T00:00:00Z"}`,
			nil,
			400,
		},
		{"null", `{"coordinates":null,"recorded_at":"2026-11-15T00:00:00Z"}`, nil, 400},
		{
			"wrong time",
			`{"coordinates":{"latitude":0,"longitude":0},"recorded_at":"bad"}`,
			nil,
			400,
		},
		{
			"unknown",
			`{"coordinates":{"latitude":0,"longitude":0,"extra":1},"recorded_at":"2026-11-15T00:00:00Z"}`,
			nil,
			400,
		},
		{
			"wrong case",
			`{"Coordinates":{"latitude":0,"longitude":0},"recorded_at":"2026-11-15T00:00:00Z"}`,
			nil,
			400,
		},
		{"trailing", `{} {}`, nil, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usecase := NewMockusecase(gomock.NewController(t))
			id := uuid.New()

			if tc.status != 400 || tc.err != nil {
				usecase.EXPECT().Execute(gomock.Any(), id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, data trip.PositionData) (trip.Position, error) {
					return trip.Position{ID: 42, TripID: id, PositionData: data}, tc.err
				})
			}

			router := chi.NewRouter()
			router.Post("/{tripId}", saveposition.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))).ServeHTTP)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/"+id.String(), strings.NewReader(tc.body)))

			if response.Code != tc.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}

			if strings.Contains(response.Body.String(), "secret") {
				t.Fatal("internal details leaked")
			}

			if tc.status == 201 && !strings.Contains(response.Body.String(), `"id":42`) {
				t.Fatal("missing position")
			}
		})
	}
}
