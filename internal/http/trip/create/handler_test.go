package create_test

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	trip "github.com/n1ckerr0r/avito-labs/internal/domain/trip"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/trip/create"
	"github.com/n1ckerr0r/avito-labs/internal/model"
	usecase "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
)

const validBody = `{"user_id":"11111111-1111-1111-1111-111111111111","driver_id":"22222222-2222-2222-2222-222222222222","start_point":{"latitude":1,"longitude":1},"end_point":{"latitude":2,"longitude":2},"price":500}`

func TestHandler_Handle(t *testing.T) {
	tripID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	key := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	hash := sha256.Sum256([]byte(validBody))
	input := usecase.Input{
		UserID:     uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		DriverID:   uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		StartPoint: trip.Coordinates{Latitude: 1, Longitude: 1},
		EndPoint:   trip.Coordinates{Latitude: 2, Longitude: 2},
		Price:      500, IdempotencyKey: &key, RequestHash: hash[:],
	}
	result := model.Trip{ID: tripID, Status: string(trip.StatusActive)}

	tests := []struct {
		name, body             string
		callUsecase            bool
		result                 model.Trip
		created                bool
		err                    error
		wantStatus             int
		wantCode, wantLocation string
	}{
		{name: "created", body: validBody, callUsecase: true, result: result, created: true, wantStatus: http.StatusCreated, wantLocation: "/api/v1/trips/" + tripID.String()},
		{name: "idempotent replay", body: validBody, callUsecase: true, result: result, wantStatus: http.StatusOK},
		{name: "invalid body", body: `{}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "domain error", body: validBody, callUsecase: true, err: trip.ErrDriverBusy, wantStatus: http.StatusConflict, wantCode: "driver_busy"},
		{name: "repository error hides details", body: validBody, callUsecase: true, err: assert.AnError, wantStatus: http.StatusInternalServerError, wantCode: "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			uc := NewMocktripCreator(ctrl)
			metrics := NewMockmetrics(ctrl)
			logger := NewMocklogger(ctrl)
			if tt.callUsecase {
				uc.EXPECT().Execute(gomock.Any(), input).Return(usecase.Result{Trip: tt.result, Created: tt.created}, tt.err)
			}
			if tt.created {
				metrics.EXPECT().IncCreated(gomock.Any())
				logger.EXPECT().InfoContext(gomock.Any(), "trip created", "trip_id", tripID)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			create.New(uc, metrics, logger).Handle(rec, req, api.CreateTripParams{IdempotencyKey: &key})

			assert.Equal(t, tt.wantStatus, rec.Code)
			assert.Equal(t, tt.wantLocation, rec.Header().Get("Location"))
			if tt.wantCode != "" {
				var problem api.Problem
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&problem))
				assert.Equal(t, tt.wantCode, problem.Code)
				if tt.err != nil && tt.wantStatus == http.StatusInternalServerError {
					assert.NotContains(t, rec.Body.String(), tt.err.Error())
				}
			}
		})
	}
}
