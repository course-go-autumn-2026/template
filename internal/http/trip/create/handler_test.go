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
	createhandler "github.com/George1487/MalyginGeorge/internal/http/trip/create"
)

func TestHandler_Handle(t *testing.T) {
	t.Parallel()
	type mocks struct {
		usecase *MockcreateUsecase
	}

	type in struct {
		body  string
		setup func(m mocks)
	}

	type out struct {
		assert func(t *testing.T, rec *httptest.ResponseRecorder)
	}

	tripID := uuid.New()
	userID := uuid.New()
	driverID := uuid.New()

	validBody := `{
		"user_id": "` + userID.String() + `",
		"driver_id": "` + driverID.String() + `",
		"start_point": {"latitude": 55.7558, "longitude": 37.6173},
		"end_point": {"latitude": 55.7522, "longitude": 37.6155},
		"price": 50000
	}`

	createdTrip := trip.Trip{
		ID:       tripID,
		UserID:   userID,
		DriverID: driverID,
		StartPoint: trip.Coordinates{
			Latitude:  55.7558,
			Longitude: 37.6173,
		},
		EndPoint: trip.Coordinates{
			Latitude:  55.7522,
			Longitude: 37.6155,
		},
		Price:     50000,
		Status:    trip.StatusActive,
		StartedAt: time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name string
		in   in
		out  out
	}{
		{
			name: "success 201",
			in: in{
				body: validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(createdTrip, nil)
				},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusCreated, rec.Code)
					require.Equal(t, "/api/v1/trips/"+tripID.String(), rec.Header().Get("Location"))
					require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

					var resp api.Trip
					err := json.NewDecoder(rec.Body).Decode(&resp)
					require.NoError(t, err)
					require.Equal(t, tripID, resp.Id)
				},
			},
		},
		{
			name: "invalid request body 400",
			in: in{
				body:  `{"invalid": true}`,
				setup: func(m mocks) {},
			},
			out: out{
				assert: func(t *testing.T, rec *httptest.ResponseRecorder) {
					require.Equal(t, http.StatusBadRequest, rec.Code)
					var p api.Problem
					err := json.NewDecoder(rec.Body).Decode(&p)
					require.NoError(t, err)
					require.Equal(t, "invalid_request", p.Code)
				},
			},
		},
		{
			name: "usecase internal error 500",
			in: in{
				body: validBody,
				setup: func(m mocks) {
					m.usecase.EXPECT().
						Execute(gomock.Any(), gomock.Any()).
						Return(trip.Trip{}, errors.New("save failure"))
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
				usecase: NewMockcreateUsecase(ctrl),
			}
			tt.in.setup(m)

			h := createhandler.New(m.usecase)
			req := httptest.NewRequest(http.MethodPost, "/trips", bytes.NewBufferString(tt.in.body))
			rec := httptest.NewRecorder()

			h.Handle(rec, req)

			tt.out.assert(t, rec)
		})
	}
}
