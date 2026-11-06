package finish_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	handler "github.com/toooriii/tori/internal/http/trip/finish"
	"github.com/toooriii/tori/internal/model"
)

func newHandler(uc handler.UseCase) *handler.Handler {
	return handler.New(uc)
}

func TestFinishTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	id := uuid.New()
	now := time.Now().UTC()
	uc.EXPECT().Do(gomock.Any(), id, gomock.Any()).Return(model.Trip{
		ID: id, Status: trip.StatusCompleted, FinishedAt: &now,
	}, nil)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	rec := httptest.NewRecorder()
	h.FinishTrip(rec, req, id)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestFinishTrip_AlreadyCompleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	id := uuid.New()
	uc.EXPECT().Do(gomock.Any(), id, gomock.Any()).Return(model.Trip{}, trip.ErrTripCompleted)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	rec := httptest.NewRecorder()
	h.FinishTrip(rec, req, id)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("trip_completed")) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestFinishTrip_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	id := uuid.New()
	uc.EXPECT().Do(gomock.Any(), id, gomock.Any()).Return(model.Trip{}, trip.ErrTripNotFound)

	h := newHandler(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
	rec := httptest.NewRecorder()
	h.FinishTrip(rec, req, id)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
