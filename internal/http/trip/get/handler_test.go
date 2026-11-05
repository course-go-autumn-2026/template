package get_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/http/trip/get"
	"github.com/xotapy-main/courseGo/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	id := uuid.New()

	t.Run("200 ok", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), id).Return(&model.Trip{ID: id}, nil)

		h := get.New(uc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, id)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("404 not found", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), id).Return(nil, trip.ErrTripNotFound)

		h := get.New(uc)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, id)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})
}
