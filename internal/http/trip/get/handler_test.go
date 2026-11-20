package get_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/get"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	expected := model.Trip{ID: id, UserID: uuid.New(), DriverID: uuid.New(), Price: 500, Status: string(trip.StatusActive)}
	tests := []struct {
		name   string
		setup  func(*gomock.Controller) *get.Handler
		status int
		code   string
	}{
		{name: "found", setup: func(ctrl *gomock.Controller) *get.Handler {
			uc := NewMocktripGetter(ctrl)
			uc.EXPECT().Execute(gomock.Any(), id).Return(expected, nil)
			return get.New(uc)
		}, status: http.StatusOK},
		{name: "not found", setup: func(ctrl *gomock.Controller) *get.Handler {
			uc := NewMocktripGetter(ctrl)
			uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, trip.ErrTripNotFound)
			return get.New(uc)
		}, status: http.StatusNotFound, code: "trip_not_found"},
		{name: "unexpected error", setup: func(ctrl *gomock.Controller) *get.Handler {
			uc := NewMocktripGetter(ctrl)
			uc.EXPECT().Execute(gomock.Any(), id).Return(model.Trip{}, assert.AnError)
			return get.New(uc)
		}, status: http.StatusInternalServerError, code: "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup(gomock.NewController(t))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+id.String(), nil)
			rec := httptest.NewRecorder()
			h.Handle(rec, req, id)
			require.Equal(t, tt.status, rec.Code)
			if tt.code != "" {
				var problem api.Problem
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
				assert.Equal(t, tt.code, problem.Code)
				if tt.status == http.StatusInternalServerError {
					assert.NotContains(t, rec.Body.String(), assert.AnError.Error())
				}
				return
			}
			var got api.Trip
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
			assert.Equal(t, expected.ID, got.Id)
			assert.Equal(t, expected.UserID, got.UserId)
			assert.Equal(t, expected.DriverID, got.DriverId)
			assert.Equal(t, expected.Price, got.Price)
			assert.Equal(t, api.TripStatus(expected.Status), got.Status)
		})
	}
}
