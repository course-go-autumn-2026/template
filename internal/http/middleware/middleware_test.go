package middleware_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/XrystikLK/template/internal/http/middleware"
)

type mockMetricsRecorder struct {
	recordedDuration bool
	recordedStatus   string
	recordedRoute    string
	activeRequests   int64
	activeRoute      string
}

func (m *mockMetricsRecorder) RecordRequestDuration(_ context.Context, _ float64, _, route, statusCode string) {
	m.recordedDuration = true
	m.recordedRoute = route
	m.recordedStatus = statusCode
}

func (m *mockMetricsRecorder) AddActiveRequests(_ context.Context, delta int64, _, route string) {
	m.activeRequests += delta
	m.activeRoute = route
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}
	if logEntry["level"] != "ERROR" {
		t.Fatalf("expected log level ERROR, got %v", logEntry["level"])
	}
}

func TestLogging(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	r := chi.NewRouter()
	r.Use(middleware.Logging(logger))
	r.Get("/test/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test/42", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var logEntry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("failed to parse log json: %v", err)
	}
	if logEntry["route"] != "/test/{id}" {
		t.Fatalf("expected route template /test/{id}, got %v", logEntry["route"])
	}
	if logEntry["method"] != "GET" {
		t.Fatalf("expected method GET, got %v", logEntry["method"])
	}
}

func TestTracingAndMetrics(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	tracer := tp.Tracer("test")

	recorder := &mockMetricsRecorder{}

	r := chi.NewRouter()
	r.Use(middleware.Tracing(tracer))
	r.Use(middleware.Metrics(recorder))
	r.Get("/items/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/items/100", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if !recorder.recordedDuration {
		t.Fatalf("expected duration to be recorded")
	}

	if recorder.activeRequests != 0 {
		t.Fatalf("expected activeRequests to be 0 after request, got %d", recorder.activeRequests)
	}
}

func TestMiddlewareChain_Panic_RecordsTelemetryAndRecovers(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tracer := tp.Tracer("test")

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	recorder := &mockMetricsRecorder{}

	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Tracing(tracer))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.Metrics(recorder))

	r.Get("/api/v1/trips/{tripId}", func(w http.ResponseWriter, r *http.Request) {
		panic("database connection failure")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/11111111-2222-3333-4444-555555555555", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))

	assert.True(t, recorder.recordedDuration)
	assert.Equal(t, "500", recorder.recordedStatus)
	assert.Equal(t, "/api/v1/trips/{tripId}", recorder.recordedRoute)
	assert.Equal(t, int64(0), recorder.activeRequests)
	assert.Equal(t, "/api/v1/trips/{tripId}", recorder.activeRoute)

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	span := spans[len(spans)-1]
	assert.Equal(t, codes.Error, span.Status.Code)

	var hasRoute, hasErrorType bool
	for _, attr := range span.Attributes {
		if attr.Key == "http.route" && attr.Value.AsString() == "/api/v1/trips/{tripId}" {
			hasRoute = true
		}
		if attr.Key == "error.type" && attr.Value.AsString() == "internal_error" {
			hasErrorType = true
		}
	}
	assert.True(t, hasRoute, "span must contain http.route attribute")
	assert.True(t, hasErrorType, "span must contain error.type attribute")

	logStr := logBuf.String()
	assert.Contains(t, logStr, "panic recovered")
	assert.Contains(t, logStr, "request handled")
	assert.Contains(t, logStr, "/api/v1/trips/{tripId}")
}
