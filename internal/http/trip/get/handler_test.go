package get_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	handler "github.com/toooriii/tori/internal/http/trip/get"
	"github.com/toooriii/tori/internal/model"
)

func newHandler(uc handler.UseCase) *handler.Handler {
	return handler.New(uc)
}

func TestGetTrip_Found(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	id := uuid.New()
	uc.EXPECT().Do(gomock.Any(), id).Return(model.Trip{ID: id, Status: trip.StatusActive}, nil)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rec := httptest.NewRecorder()
	h.GetTrip(rec, req, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestGetTrip_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	id := uuid.New()
	uc.EXPECT().Do(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rec := httptest.NewRecorder()
	h.GetTrip(rec, req, id)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("trip_not_found")) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}
