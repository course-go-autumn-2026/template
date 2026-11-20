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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	poscreate "github.com/George1487/MalyginGeorge/internal/http/trip/position/create"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	type mocks struct {
		usecase *Mockusecase
	}

	type in struct {
		tripID api.TripId
		body   string
		setup  func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	tripID := uuid.New()
	recordedAt := time.Date(2026, 8, 24, 12, 0, 5, 0, time.UTC)

	validBody := `{
		"coordinates": {"latitude": 59.9350, "longitude": 30.3350},
		"recorded_at": "` + recordedAt.Format(time.RFC3339) + `"
	}`

	createdPosition := &trip.Position{
		ID:     42,
		TripID: tripID,
		Coordinates: trip.Coordinates{
			Latitude:  59.9350,
			Longitude: 30.3350,
		},
		RecordedAt: recordedAt,
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success: 201 Created with saved position",
			in: in{
				tripID: tripID,
				body:   validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(createdPosition, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusCreated, rec.Code)
					var resp api.TripPosition
					err := json.Unmarshal(rec.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.InDelta(t, 59.9350, resp.Coordinates.Latitude, 0.0001)
					require.InDelta(t, 30.3350, resp.Coordinates.Longitude, 0.0001)
				},
			},
		},
		{
			name: "error: invalid JSON returns 400 Bad Request",
			in: in{
				tripID: tripID,
				body:   `{invalid json`,
				setup: func(m mocks) {
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, rec.Code)
				},
			},
		},
		{
			name: "error: invalid coordinates returns 400 Bad Request",
			in: in{
				tripID: tripID,
				body: `{
					"coordinates": {"latitude": 159.9350, "longitude": 30.3350},
					"recorded_at": "` + recordedAt.Format(time.RFC3339) + `"
				}`,
				setup: func(m mocks) {
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, rec.Code)
				},
			},
		},
		{
			name: "error: trip completed returns 409 Conflict",
			in: in{
				tripID: tripID,
				body:   validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(nil, trip.ErrTripCompleted)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusConflict, rec.Code)
				},
			},
		},
		{
			name: "error: trip not found returns 404 Not Found",
			in: in{
				tripID: tripID,
				body:   validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(nil, trip.ErrTripNotFound)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusNotFound, rec.Code)
				},
			},
		},
		{
			name: "error: unexpected usecase error returns 500 Internal Server Error",
			in: in{
				tripID: tripID,
				body:   validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("db connection lost"))
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, rec.Code)
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			m := mocks{
				usecase: NewMockusecase(ctrl),
			}
			tt.in.setup(m)

			handler := poscreate.New(m.usecase)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tt.in.tripID.String()+"/positions", bytes.NewBufferString(tt.in.body))
			rec := httptest.NewRecorder()

			handler.Handle(rec, req, tt.in.tripID)

			tt.out.assert(t, rec)
		})
	}
}
