package list_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/http/position/list"
	"github.com/xotapy-main/courseGo/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	tripID := uuid.New()

	t.Run("200 ok", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), tripID).Return([]model.Position{
			{ID: 1, TripID: tripID, RecordedAt: time.Now().UTC()},
		}, nil)

		h := list.New(uc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String()+"/positions", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("404 trip not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), tripID).Return(nil, domaintrip.ErrTripNotFound)

		h := list.New(uc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String()+"/positions", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
