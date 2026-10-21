package httpapi

import (
	"bytes"
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
	"go.uber.org/mock/gomock"

	domain "github.com/whxtelyy/template/internal/domain/trip"
	api "github.com/whxtelyy/template/internal/generated"
	"github.com/whxtelyy/template/internal/usecase/trip/create"
)

func testHandler(t *testing.T) (*Handler, *MockcreateUseCase, *MockgetUseCase, *MockfinishUseCase) {
	t.Helper()
	ctrl := gomock.NewController(t)
	c, g, f := NewMockcreateUseCase(ctrl), NewMockgetUseCase(ctrl), NewMockfinishUseCase(ctrl)
	ready := NewMockreadiness(ctrl)
	return NewHandler(c, g, f, ready, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second), c, g, f
}

func TestCreateTripResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		replay bool
		status int
	}{
		{"created", false, http.StatusCreated}, {"replayed", true, http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, c, _, _ := testHandler(t)
			id := uuid.New()
			c.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(create.Result{Trip: domain.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Status: domain.StatusActive}, Replayed: tc.replay}, nil)
			body, _ := json.Marshal(api.TripData{UserId: uuid.New(), DriverId: uuid.New(), StartPoint: api.Coordinates{}, EndPoint: api.Coordinates{}, Price: 100})
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(body))
			w := httptest.NewRecorder()
			h.CreateTrip(w, req, api.CreateTripParams{})
			if w.Code != tc.status || w.Header().Get("Location") != "/api/v1/trips/"+id.String() {
				t.Fatalf("status %d, headers %v, body %s", w.Code, w.Header(), w.Body.String())
			}
		})
	}
}

func TestCreateTripMalformedBody(t *testing.T) {
	h, _, _, _ := testHandler(t)
	w := httptest.NewRecorder()
	h.CreateTrip(w, httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader("{")), api.CreateTripParams{})
	if w.Code != http.StatusBadRequest || w.Header().Get("Content-Type") != "application/problem+json" || !strings.Contains(w.Body.String(), `"code":"invalid_request"`) {
		t.Fatalf("response: %d %s", w.Code, w.Body.String())
	}
}

func TestGetTripErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"missing", domain.ErrTripNotFound, http.StatusNotFound, "trip_not_found"},
		{"internal", errors.New("secret database credential"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _, g, _ := testHandler(t)
			id := uuid.New()
			g.EXPECT().Execute(gomock.Any(), id).Return(domain.Trip{}, tc.err)
			w := httptest.NewRecorder()
			h.GetTrip(w, httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil), id)
			if w.Code != tc.status || !strings.Contains(w.Body.String(), `"code":"`+tc.code+`"`) || strings.Contains(w.Body.String(), "secret database") {
				t.Fatalf("response: %d %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestFinishTripCompleted(t *testing.T) {
	h, _, _, f := testHandler(t)
	id := uuid.New()
	f.EXPECT().Execute(gomock.Any(), id).Return(domain.Trip{}, domain.ErrTripCompleted)
	w := httptest.NewRecorder()
	h.FinishTrip(w, httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil).WithContext(context.Background()), id)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"code":"trip_completed"`) {
		t.Fatalf("response: %d %s", w.Code, w.Body.String())
	}
}
