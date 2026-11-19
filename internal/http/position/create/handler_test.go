package create_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/http/position/create"
	"github.com/xotapy-main/courseGo/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	tripID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	validBody := []byte(`{"coordinates":{"latitude":59.935,"longitude":30.335},"recorded_at":"` + now.Format(time.RFC3339) + `"}`)

	t.Run("201 created", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)

		uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(&model.Position{
			ID:          42,
			TripID:      tripID,
			Coordinates: model.Point{Latitude: 59.935, Longitude: 30.335},
			RecordedAt:  now,
		}, nil)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/positions", bytes.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("400 invalid json", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/positions", bytes.NewReader([]byte(`{"bad":1}`)))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)
		require.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("409 trip completed", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, domaintrip.ErrTripCompleted)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/positions", bytes.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)
		require.Equal(t, http.StatusConflict, rec.Code)
	})
}
