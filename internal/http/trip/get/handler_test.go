package get

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	tripdomain "github.com/murmyauuu/template/internal/domain/trip"
)

type fakeUsecase struct {
	trip tripdomain.Trip
	err  error
}

func (f *fakeUsecase) Execute(ctx context.Context, id uuid.UUID) (tripdomain.Trip, error) {
	return f.trip, f.err
}

func TestHandlerGetSuccess(t *testing.T) {
	id := uuid.New()
	fu := &fakeUsecase{trip: tripdomain.Trip{Id: id, Status: tripdomain.StatusActive}}
	h := &Handler{Usecase: fu}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rr := httptest.NewRecorder()
	h.GetTrip(rr, req, id)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestHandlerGetNotFound(t *testing.T) {
	id := uuid.New()
	fu := &fakeUsecase{err: tripdomain.ErrTripNotFound}
	h := &Handler{Usecase: fu}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
	rr := httptest.NewRecorder()
	h.GetTrip(rr, req, id)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}
