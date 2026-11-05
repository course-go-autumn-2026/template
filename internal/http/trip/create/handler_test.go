package create_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/domain/trip"
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/http/trip/create"
	"github.com/xotapy-main/courseGo/internal/model"
)

func TestHandler_ServeHTTP(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	validBody := []byte(`{
		"user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point":{"latitude":59.93,"longitude":30.31},
		"end_point":{"latitude":59.92,"longitude":30.36},
		"price":1450
	}`)

	t.Run("201 created", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)

		expected := &model.Trip{
			ID:        uuid.New(),
			UserID:    uuid.MustParse("5cb72c04-7650-45c9-a79b-bcdba0631e0c"),
			DriverID:  uuid.MustParse("8860b315-ec86-42eb-a17c-7c163d721ff5"),
			Price:     1450,
			Status:    model.StatusActive,
			StartedAt: time.Now().UTC(),
		}
		uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(expected, nil)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Code)
	})

	t.Run("400 invalid body", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader([]byte(`{"unknown_field":123}`)))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusBadRequest, rec.Code)
		var prob api.Problem
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &prob))
		require.Equal(t, "invalid_request", prob.Code)
	})

	t.Run("409 driver busy", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, trip.ErrDriverBusy)

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusConflict, rec.Code)
	})

	t.Run("500 internal error hides details", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		uc := NewMockusecase(ctrl)
		uc.EXPECT().Execute(gomock.Any(), gomock.Any()).Return(nil, errors.New("secret sql failure"))

		h := create.New(uc, logger, nil)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(validBody))
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)
		require.Equal(t, http.StatusInternalServerError, rec.Code)
		require.NotContains(t, rec.Body.String(), "secret sql failure")
	})
}
