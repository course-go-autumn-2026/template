package create_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/eternal-despair/template/internal/domain/trip"
	api "github.com/eternal-despair/template/internal/generated"
	createHandler "github.com/eternal-despair/template/internal/http/trip/create"
	"github.com/eternal-despair/template/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	validUser := uuid.New()
	validDriver := uuid.New()
	tripID := uuid.New()
	now := time.Now().UTC()

	validTrip := &model.Trip{
		ID:        tripID,
		UserID:    validUser,
		DriverID:  validDriver,
		Price:     350,
		Status:    trip.StatusActive,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	bodyMap := map[string]any{
		"user_id":   validUser.String(),
		"driver_id": validDriver.String(),
		"start_point": map[string]any{
			"latitude":  55.75,
			"longitude": 37.61,
		},
		"end_point": map[string]any{
			"latitude":  55.76,
			"longitude": 37.62,
		},
		"price": 350,
	}

	t.Run("valid request returns 201 Created", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), gomock.Any()).
			Return(validTrip, nil)

		h := createHandler.New(ucMock)

		bodyBytes, _ := json.Marshal(bodyMap)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, api.CreateTripParams{})

		require.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, "/api/v1/trips/"+tripID.String(), rec.Header().Get("Location"))

		var resp api.Trip
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, tripID, resp.Id)
	})

	t.Run("invalid body returns 400 invalid_request", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		h := createHandler.New(ucMock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader([]byte("{invalid json")))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, api.CreateTripParams{})

		require.Equal(t, http.StatusBadRequest, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "invalid_request", prob.Code)
	})

	t.Run("driver busy returns 409 driver_busy", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), gomock.Any()).
			Return(nil, trip.ErrDriverBusy)

		h := createHandler.New(ucMock)

		bodyBytes, _ := json.Marshal(bodyMap)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, api.CreateTripParams{})

		require.Equal(t, http.StatusConflict, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "driver_busy", prob.Code)
	})

	t.Run("internal error returns 500 without details", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("sql: syntax error in query INSERT INTO trips"))

		h := createHandler.New(ucMock)

		bodyBytes, _ := json.Marshal(bodyMap)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()

		h.Handle(rec, req, api.CreateTripParams{})

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "internal_error", prob.Code)
		require.NotNil(t, prob.Detail)
		assert.NotContains(t, *prob.Detail, "INSERT INTO")
	})
}
