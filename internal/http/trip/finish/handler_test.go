package finish_test

import (
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
	finishHandler "github.com/eternal-despair/template/internal/http/trip/finish"
	"github.com/eternal-despair/template/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	tripID := uuid.New()
	finishTime := time.Now().UTC()
	sampleFinishedTrip := &model.Trip{
		ID:         tripID,
		UserID:     uuid.New(),
		DriverID:   uuid.New(),
		Status:     trip.StatusCompleted,
		Price:      500,
		StartedAt:  finishTime.Add(-20 * time.Minute),
		FinishedAt: &finishTime,
	}

	t.Run("valid finish returns 200 OK", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(sampleFinishedTrip, nil)

		h := finishHandler.New(ucMock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp api.Trip
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Equal(t, tripID, resp.Id)
		assert.Equal(t, api.TripStatus(trip.StatusCompleted), resp.Status)
	})

	t.Run("already completed returns 409 trip_completed", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(nil, trip.ErrTripCompleted)

		h := finishHandler.New(ucMock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		require.Equal(t, http.StatusConflict, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "trip_completed", prob.Code)
	})

	t.Run("trip not found returns 404 trip_not_found", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(nil, trip.ErrTripNotFound)

		h := finishHandler.New(ucMock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		require.Equal(t, http.StatusNotFound, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "trip_not_found", prob.Code)
	})

	t.Run("internal error returns 500 without details", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		ucMock := NewMockusecase(ctrl)

		ucMock.EXPECT().
			Execute(gomock.Any(), tripID).
			Return(nil, errors.New("database connection broken"))

		h := finishHandler.New(ucMock)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
		rec := httptest.NewRecorder()

		h.Handle(rec, req, tripID)

		require.Equal(t, http.StatusInternalServerError, rec.Code)
		var prob api.Problem
		err := json.Unmarshal(rec.Body.Bytes(), &prob)
		require.NoError(t, err)
		assert.Equal(t, "internal_error", prob.Code)
	})
}
