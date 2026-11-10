package create

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/murmyauuu/template/api"
	tripdomain "github.com/murmyauuu/template/internal/domain/trip"
	uc "github.com/murmyauuu/template/internal/usecase/trip/create"
)

const (
	latitudeKey  = "latitude"
	longitudeKey = "longitude"
)

type fakeUsecase struct {
	out uc.Output
	err error
}

func (f *fakeUsecase) Execute(ctx context.Context, in uc.Input, key *uuid.UUID) (uc.Output, error) {
	return f.out, f.err
}

func TestHandlerCreateSuccess(t *testing.T) {
	id := uuid.New()
	fu := &fakeUsecase{out: uc.Output{Trip: tripdomain.Trip{Id: id, Status: tripdomain.StatusActive}, Fresh: true}}
	h := &Handler{Usecase: fu}

	body := map[string]any{
		"user_id":     uuid.New().String(),
		"driver_id":   uuid.New().String(),
		"start_point": map[string]float64{latitudeKey: 1, longitudeKey: 2},
		"end_point":   map[string]float64{latitudeKey: 3, longitudeKey: 4},
		"price":       100,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateTrip(rr, req, api.CreateTripParams{})
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rr.Code)
	}
}

func TestHandlerCreateInvalidBody(t *testing.T) {
	h := &Handler{Usecase: &fakeUsecase{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateTrip(rr, req, api.CreateTripParams{})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestHandlerCreateDriverBusy(t *testing.T) {
	fu := &fakeUsecase{err: tripdomain.ErrDriverBusy}
	h := &Handler{Usecase: fu}
	body := map[string]any{
		"user_id":     uuid.New().String(),
		"driver_id":   uuid.New().String(),
		"start_point": map[string]float64{latitudeKey: 1, longitudeKey: 2},
		"end_point":   map[string]float64{latitudeKey: 3, longitudeKey: 4},
		"price":       100,
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateTrip(rr, req, api.CreateTripParams{})
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}
