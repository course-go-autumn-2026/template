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
	gethandler "github.com/George1487/MalyginGeorge/internal/http/trip/get"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	type mocks struct {
		usecase *MockgetUsecase
	}

	type in struct {
		tripID api.TripId
		setup  func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	tripID := uuid.New()
	sampleTrip := trip.Trip{
		ID:        tripID,
		UserID:    uuid.New(),
		DriverID:  uuid.New(),
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success 200",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
						Return(sampleTrip, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusOK, rec.Code)
					require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

					var resp api.Trip
					err := json.NewDecoder(rec.Body).Decode(&resp)
					require.NoError(t, err)
					require.Equal(t, tripID, resp.Id)
				},
			},
		},
		{
			name: "not found 404",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
						Return(trip.Trip{}, trip.ErrTripNotFound)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusNotFound, rec.Code)
					var p api.Problem
					err := json.NewDecoder(rec.Body).Decode(&p)
					require.NoError(t, err)
					require.Equal(t, "trip_not_found", p.Code)
				},
			},
		},
		{
			name: "usecase error 500",
			in: in{
				tripID: tripID,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), tripID).
						Return(trip.Trip{}, errors.New("internal failure"))
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusInternalServerError, rec.Code)
					var p api.Problem
					err := json.NewDecoder(rec.Body).Decode(&p)
					require.NoError(t, err)
					require.Equal(t, "internal_error", p.Code)
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
				usecase: NewMockgetUsecase(ctrl),
			}
			tt.in.setup(m)

			h := gethandler.New(m.usecase)
			req := httptest.NewRequest(http.MethodGet, "/trips/"+tt.in.tripID.String(), nil)
			rec := httptest.NewRecorder()

			h.Handle(rec, req, tt.in.tripID)

			tt.out.assert(t, rec)
		})
	}
}
