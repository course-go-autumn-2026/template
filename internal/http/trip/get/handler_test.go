package get_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/mock/gomock"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	getHandler "github.com/XrystikLK/template/internal/http/trip/get"
	"github.com/XrystikLK/template/internal/model"
)

func TestGetTripHandler(t *testing.T) {
	tripID := uuid.New()

	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUC := NewMockuseCase(ctrl)
		now := time.Now().UTC()
		returnedModel := &model.Trip{
			ID:        tripID,
			Price:     400,
			Status:    model.StatusActive,
			StartedAt: now,
		}

		mockUC.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(returnedModel, nil)

		h := getHandler.New(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUC := NewMockuseCase(ctrl)
		mockUC.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(nil, domainTrip.ErrTripNotFound)

		h := getHandler.New(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("internal server error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockUC := NewMockuseCase(ctrl)
		mockUC.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(nil, errors.New("read error"))

		h := getHandler.New(mockUC)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", rec.Code)
		}
	})
}
