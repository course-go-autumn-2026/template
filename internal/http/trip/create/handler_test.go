package create_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	"github.com/toooriii/tori/internal/domain/trip"
	api "github.com/toooriii/tori/internal/generated"
	handler "github.com/toooriii/tori/internal/http/trip/create"
	"github.com/toooriii/tori/internal/model"
	usecase "github.com/toooriii/tori/internal/usecase/trip/create"
)

func validBody() string {
	return `{"user_id":"` + uuid.NewString() + `","driver_id":"` + uuid.NewString() + `","start_point":{"latitude":1,"longitude":1},"end_point":{"latitude":2,"longitude":2},"price":100}`
}

func TestCreateTrip_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	uc.EXPECT().Do(gomock.Any(), gomock.Any()).Return(usecase.Output{
		Trip: model.Trip{ID: uuid.New(), Status: trip.StatusActive},
	}, nil)

	h := handler.New(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validBody()))
	rec := httptest.NewRecorder()
	h.CreateTrip(rec, req, api.CreateTripParams{})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}

func TestCreateTrip_DriverBusy(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	uc.EXPECT().Do(gomock.Any(), gomock.Any()).Return(usecase.Output{}, trip.ErrDriverBusy)

	h := handler.New(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validBody()))
	rec := httptest.NewRecorder()
	h.CreateTrip(rec, req, api.CreateTripParams{})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestCreateTrip_InvalidBody(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	h := handler.New(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()
	h.CreateTrip(rec, req, api.CreateTripParams{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("invalid_request")) {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestCreateTrip_RepoErrorDoesNotLeak(t *testing.T) {
	ctrl := gomock.NewController(t)
	uc := NewMockUseCase(ctrl)
	uc.EXPECT().Do(gomock.Any(), gomock.Any()).Return(usecase.Output{}, errors.New("db exploded"))

	h := handler.New(uc)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(validBody()))
	rec := httptest.NewRecorder()
	h.CreateTrip(rec, req, api.CreateTripParams{})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if bytes.Contains(rec.Body.Bytes(), []byte("db exploded")) {
		t.Fatalf("internal details leaked: %s", rec.Body.String())
	}
}
