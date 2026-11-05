package finish_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/http/trip/finish"
	"github.com/xotapy-main/courseGo/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	id := uuid.New()

	t.Run("200 completed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), id).Return(&model.Trip{ID: id, Status: model.StatusCompleted}, nil)

		h := finish.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, id)
		require.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("409 already completed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), id).Return(nil, trip.ErrTripCompleted)

		h := finish.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, id)
		require.Equal(t, http.StatusConflict, rec.Code)
	})
}
