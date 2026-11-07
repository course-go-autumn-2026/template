package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/XrystikLK/template/internal/generated"
	"github.com/XrystikLK/template/internal/handler"
	"github.com/XrystikLK/template/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	createTripErr       error
	getTripErr          error
	finishTripErr       error
	addStatusHistoryErr error
	tripToReturn        *repository.Trip
}

func (m *mockRepo) CreateTrip(ctx context.Context, trip *repository.Trip) error {
	return m.createTripErr
}

func (m *mockRepo) GetTripByID(ctx context.Context, id uuid.UUID) (*repository.Trip, error) {
	if m.getTripErr != nil {
		return nil, m.getTripErr
	}
	return m.tripToReturn, nil
}

func (m *mockRepo) FinishTrip(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*repository.Trip, error) {
	if m.finishTripErr != nil {
		return nil, m.finishTripErr
	}
	return m.tripToReturn, nil
}

func (m *mockRepo) AddStatusHistory(ctx context.Context, history *repository.StatusHistory) error {
	return m.addStatusHistoryErr
}

type mockTxManager struct {
	doErr error
}

func (m *mockTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.doErr != nil {
		return m.doErr
	}
	return fn(ctx)
}

type mockPinger struct {
	pingErr error
}

func (m *mockPinger) Ping(ctx context.Context) error {
	return m.pingErr
}

func setupTestRouter(repo handler.TripRepository, txMgr *mockTxManager, pinger *mockPinger) http.Handler {
	h := handler.NewTripHandler(repo, txMgr, pinger)
	r := chi.NewRouter()
	api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handler.WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error(), r.URL.Path)
		},
	})
	return r
}

func TestCreateTrip_500_HidesDatabaseDetails(t *testing.T) {
	sensitiveDBError := errors.New("pq: relation \"trips\" does not exist (SQL: INSERT INTO trips ... password=secret)")
	repo := &mockRepo{createTripErr: sensitiveDBError}
	txMgr := &mockTxManager{}
	pinger := &mockPinger{}

	router := setupTestRouter(repo, txMgr, pinger)

	body := `{
		"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
		"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
		"start_point": {"latitude": 59.9398, "longitude": 30.3146},
		"end_point": {"latitude": 59.9290, "longitude": 30.3626},
		"price": 1450
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	var problem api.Problem
	err := json.Unmarshal(w.Body.Bytes(), &problem)
	require.NoError(t, err)

	assert.Equal(t, "internal_error", problem.Code)
	assert.Equal(t, int32(500), problem.Status)

	bodyStr := w.Body.String()
	assert.NotContains(t, bodyStr, "pq:")
	assert.NotContains(t, bodyStr, "relation")
	assert.NotContains(t, bodyStr, "SQL")
	assert.NotContains(t, bodyStr, "password")
	assert.NotContains(t, bodyStr, sensitiveDBError.Error())
}

func TestGetTrip_500_HidesDatabaseDetails(t *testing.T) {
	sensitiveDBError := errors.New("pq: connection refused to db-primary.internal:5432")
	repo := &mockRepo{getTripErr: sensitiveDBError}
	txMgr := &mockTxManager{}
	pinger := &mockPinger{}

	router := setupTestRouter(repo, txMgr, pinger)

	tripID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/"+tripID.String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	bodyStr := w.Body.String()
	assert.Contains(t, bodyStr, "internal_error")
	assert.NotContains(t, bodyStr, "connection refused")
	assert.NotContains(t, bodyStr, "db-primary")
}

func TestFinishTrip_500_HidesDatabaseDetails(t *testing.T) {
	sensitiveDBError := errors.New("pq: deadlock detected on update trips")
	repo := &mockRepo{finishTripErr: sensitiveDBError}
	txMgr := &mockTxManager{}
	pinger := &mockPinger{}

	router := setupTestRouter(repo, txMgr, pinger)

	tripID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trips/"+tripID.String()+"/finish", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))

	bodyStr := w.Body.String()
	assert.Contains(t, bodyStr, "internal_error")
	assert.NotContains(t, bodyStr, "deadlock detected")
}

func TestCreateTrip_RequiredFieldsValidation(t *testing.T) {
	repo := &mockRepo{}
	txMgr := &mockTxManager{}
	pinger := &mockPinger{}
	router := setupTestRouter(repo, txMgr, pinger)

	testCases := []struct {
		name          string
		body          string
		expectedCode  int
		expectedError string
	}{
		{
			name: "missing price",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626}
			}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "price is required",
		},
		{
			name: "missing start_point.latitude",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 500
			}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "start_point.latitude is required",
		},
		{
			name: "missing user_id",
			body: `{
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 500
			}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "user_id is required",
		},
		{
			name: "missing end_point",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"price": 500
			}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "end_point is required",
		},
		{
			name: "trailing extra json object",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 500
			} {"extra": 123}`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "unexpected data after JSON object",
		},
		{
			name: "trailing garbage text",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 59.9398, "longitude": 30.3146},
				"end_point": {"latitude": 59.9290, "longitude": 30.3626},
				"price": 500
			} some_garbage`,
			expectedCode:  http.StatusBadRequest,
			expectedError: "unexpected data after JSON object",
		},
		{
			name: "price zero is valid",
			body: `{
				"user_id": "5cb72c04-7650-45c9-a79b-bcdba0631e0c",
				"driver_id": "8860b315-ec86-42eb-a17c-7c163d721ff5",
				"start_point": {"latitude": 0.0, "longitude": 0.0},
				"end_point": {"latitude": 10.0, "longitude": 10.0},
				"price": 0
			}`,
			expectedCode: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/trips", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedCode, w.Code)
			if tc.expectedError != "" {
				assert.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
				assert.Contains(t, w.Body.String(), tc.expectedError)
				assert.Contains(t, w.Body.String(), "invalid_request")
			}
		})
	}
}

func TestRouter_InvalidParamFormat_ReturnsProblemJSON(t *testing.T) {
	repo := &mockRepo{}
	txMgr := &mockTxManager{}
	pinger := &mockPinger{}
	router := setupTestRouter(repo, txMgr, pinger)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/not-a-valid-uuid", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.True(t, strings.HasPrefix(w.Header().Get("Content-Type"), "application/problem+json"))

	var p api.Problem
	err := json.Unmarshal(w.Body.Bytes(), &p)
	require.NoError(t, err)
	assert.Equal(t, "invalid_request", p.Code)
	assert.Equal(t, int32(400), p.Status)
}

func TestReady_Handler(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		router := setupTestRouter(&mockRepo{}, &mockTxManager{}, &mockPinger{pingErr: nil})
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "ok")
	})

	t.Run("database unavailable", func(t *testing.T) {
		router := setupTestRouter(&mockRepo{}, &mockTxManager{}, &mockPinger{pingErr: errors.New("timeout")})
		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
		assert.Contains(t, w.Body.String(), "unavailable")
	})
}
