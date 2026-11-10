package finish

import (
	"bytes"
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

func TestHandlerFinishSuccess(t *testing.T) {
	id := uuid.New()
	fu := &fakeUsecase{trip: tripdomain.Trip{Id: id, Status: tripdomain.StatusCompleted}}
	h := &Handler{Usecase: fu}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", bytes.NewReader(nil))
	rr := httptest.NewRecorder()
	h.FinishTrip(rr, req, id)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
}

func TestHandlerFinishCompleted(t *testing.T) {
	id := uuid.New()
	fu := &fakeUsecase{err: tripdomain.ErrTripCompleted}
	h := &Handler{Usecase: fu}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", bytes.NewReader(nil))
	rr := httptest.NewRecorder()
	h.FinishTrip(rr, req, id)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}
