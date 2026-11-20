package finish_test

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
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/finish"
	"github.com/n1ckerr0r/avito-labs/internal/model"
)

func TestHandler_Handle(t *testing.T) {
	id := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	expected := model.Trip{ID: id, UserID: uuid.New(), Price: 500, Status: string(trip.StatusCompleted)}
	setup := func(err error) func(*gomock.Controller) *finish.Handler {
		return func(ctrl *gomock.Controller) *finish.Handler {
			uc := NewMocktripFinisher(ctrl)
			metrics := NewMockmetrics(ctrl)
			logger := NewMocklogger(ctrl)
			result := expected
			if err != nil {
				result = model.Trip{}
			} else {
				metrics.EXPECT().IncCompleted(gomock.Any())
				logger.EXPECT().InfoContext(gomock.Any(), "trip finished", "trip_id", id)
			}
			uc.EXPECT().Execute(gomock.Any(), id).Return(result, err)
			return finish.New(uc, metrics, logger)
		}
	}
	tests := []struct {
		name   string
		setup  func(*gomock.Controller) *finish.Handler
		status int
		code   string
	}{
		{"success", setup(nil), http.StatusOK, ""},
		{"already completed", setup(trip.ErrTripCompleted), http.StatusConflict, "trip_completed"},
		{"not found", setup(trip.ErrTripNotFound), http.StatusNotFound, "trip_not_found"},
		{"unexpected error", setup(assert.AnError), http.StatusInternalServerError, "internal_error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := tt.setup(gomock.NewController(t))
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+id.String()+"/finish", nil)
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
			assert.Equal(t, expected.Price, got.Price)
			assert.Equal(t, api.TripStatus(expected.Status), got.Status)
		})
	}
}
