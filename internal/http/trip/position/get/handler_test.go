package get_test

import (
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
	posget "github.com/George1487/MalyginGeorge/internal/http/trip/position/get"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	type mocks struct {
		usecase *Mockusecase
	}

	type in struct {
		tripID api.TripId
		setup  func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	tripID := uuid.New()
	recordedAt := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	positions := []trip.Position{
		{
			ID:     1,
			TripID: tripID,
			Coordinates: trip.Coordinates{
				Latitude:  59.93,
				Longitude: 30.33,
			},
			RecordedAt: recordedAt,
		},
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success: 200 OK with route positions",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
						Return(positions, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusOK, rec.Code)
					var resp api.TripPositionsResponse
					err := json.Unmarshal(rec.Body.Bytes(), &resp)
					require.NoError(t, err)
					require.Len(t, resp.Positions, 1)
					require.InDelta(t, 59.93, resp.Positions[0].Coordinates.Latitude, 0.0001)
					require.InDelta(t, 30.33, resp.Positions[0].Coordinates.Longitude, 0.0001)
				},
			},
		},
		{
			name: "error: trip not found returns 404 Not Found",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
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
			name: "error: internal error returns 500 Internal Server Error",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
						Return(nil, errors.New("db connection failure"))
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

			handler := posget.New(m.usecase)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tt.in.tripID.String()+"/positions", nil)
			rec := httptest.NewRecorder()

			handler.Handle(rec, req, tt.in.tripID)

			tt.out.assert(t, rec)
		})
	}
}
