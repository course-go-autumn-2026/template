package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	api "github.com/eternal-despair/template/internal/generated"
	"github.com/eternal-despair/template/internal/http/middleware"
)

type mockMetrics struct {
	recordedDuration bool
	activeCount      int
}

func (m *mockMetrics) RecordRequestDuration(_ context.Context, _, _ string, _ int, _ float64) {
	m.recordedDuration = true
}

func (m *mockMetrics) IncActiveRequests(_ context.Context, _, _ string) {
	m.activeCount++
}

func (m *mockMetrics) DecActiveRequests(_ context.Context, _, _ string) {
	m.activeCount--
}

func TestRecover(t *testing.T) {
	t.Parallel()

	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recoverMW := middleware.Recover(discardLogger)

	panickyHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("something went terribly wrong")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		recoverMW(panickyHandler).ServeHTTP(rec, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	var prob api.Problem
	err := json.Unmarshal(rec.Body.Bytes(), &prob)
	require.NoError(t, err)
	assert.Equal(t, "internal_error", prob.Code)
}

func TestLogging(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logMW := middleware.Logging(logger)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	rec := httptest.NewRecorder()

	logMW(okHandler).ServeHTTP(rec, req)

	output := buf.String()
	assert.Contains(t, output, `"msg":"request handled"`)
	assert.Contains(t, output, `"status":200`)
	assert.Contains(t, output, `"method":"GET"`)
	assert.Contains(t, output, `"duration_ms"`)
}

func TestMetrics(t *testing.T) {
	t.Parallel()

	metrics := &mockMetrics{}
	metricsMW := middleware.Metrics(metrics)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assert.Equal(t, 1, metrics.activeCount)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	rec := httptest.NewRecorder()

	metricsMW(okHandler).ServeHTTP(rec, req)

	assert.Equal(t, 0, metrics.activeCount)
	assert.True(t, metrics.recordedDuration)
}

func TestTracing(t *testing.T) {
	t.Parallel()

	tracer := noop.NewTracerProvider().Tracer("test")
	tracingMW := middleware.Tracing(tracer)

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips", nil)
	rec := httptest.NewRecorder()

	tracingMW(okHandler).ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
