package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/domain/trip"
	api "github.com/freyliee/template/internal/generated"
	"github.com/freyliee/template/internal/http/trip/create"
	"github.com/freyliee/template/internal/model"
)

const validBody = `{"user_id":"11111111-1111-1111-1111-111111111111","driver_id":"22222222-2222-2222-2222-222222222222","start_point":{"latitude":0,"longitude":0},"end_point":{"latitude":1,"longitude":2},"price":123}`

func TestHandler(t *testing.T) {
	cases := []struct {
		name    string
		failure error
		status  int
		code    string
	}{{"success", nil, 201, ""}, {"busy", trip.ErrDriverBusy, 409, "driver_busy"}, {"internal", errors.New("secret SQL password"), 500, "internal_error"}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			usecase := NewMockusecase(ctrl)
			id := uuid.New()
			entity := trip.Trip{ID: id, Status: trip.StatusActive, StartedAt: time.Now().UTC()}
			usecase.EXPECT().Execute(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, input model.CreateTripInput) (trip.Trip, error) {
				if input.UserID.String() != "11111111-1111-1111-1111-111111111111" || input.Price != 123 || input.EndPoint.Longitude != 2 {
					t.Fatalf("invalid input: %+v", input)
				}

				entity.Data = input

				return entity, tc.failure
			})

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(validBody))
			create.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))).ServeHTTP(w, r)

			if w.Code != tc.status {
				t.Fatalf("unexpected status: %d %s", w.Code, w.Body.String())
			}

			if tc.failure == nil {
				var body api.Trip

				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}

				if body.Id != id || body.Price != 123 || body.Status != api.Active || w.Header().Get("Location") != "/api/v1/trips/"+id.String() {
					t.Fatalf("unexpected body: %+v", body)
				}
			} else {
				var body api.Problem

				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}

				if body.Code != tc.code || w.Header().Get("Content-Type") != "application/problem+json" || strings.Contains(w.Body.String(), "secret") {
					t.Fatalf("unexpected error: %s", w.Body.String())
				}
			}
		})
	}
}

func TestInvalidBody(t *testing.T) {
	for _, body := range []string{
		"", "{", "null", "{}",
		strings.Replace(validBody, `"price":123`, `"price":1.5`, 1),
		strings.Replace(validBody, `"price":123`, `"price":123,"extra":1`, 1),
	} {
		t.Run(body, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			usecase := NewMockusecase(ctrl)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/v1/trips", strings.NewReader(body))
			create.New(usecase, slog.New(slog.NewJSONHandler(io.Discard, nil))).ServeHTTP(w, r)

			var problem api.Problem

			if err := json.Unmarshal(w.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}

			if w.Code != 400 || problem.Code != "invalid_request" {
				t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
