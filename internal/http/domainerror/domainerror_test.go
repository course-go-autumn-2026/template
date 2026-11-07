package domainerror_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	"github.com/XrystikLK/template/internal/http/domainerror"
)

func TestProblemMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "not found error",
			err:        domainTrip.ErrTripNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "trip_not_found",
		},
		{
			name:       "already completed error",
			err:        domainTrip.ErrTripCompleted,
			wantStatus: http.StatusConflict,
			wantCode:   "trip_completed",
		},
		{
			name:       "driver busy error",
			err:        domainTrip.ErrDriverBusy,
			wantStatus: http.StatusConflict,
			wantCode:   "driver_busy",
		},
		{
			name:       "validation error",
			err:        &domainTrip.ValidationError{Field: "price", Message: "cannot be negative"},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_request",
		},
		{
			name:       "unexpected error does not leak details",
			err:        errors.New("fatal database crash"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "internal_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := domainerror.Problem(tt.err)
			if status != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, status)
			}
			if body.Code != tt.wantCode {
				t.Fatalf("expected code %s, got %s", tt.wantCode, body.Code)
			}
			if tt.wantStatus == http.StatusInternalServerError && body.Detail != nil {
				t.Fatalf("expected nil detail for internal error, got %v", *body.Detail)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/123", nil)

	domainerror.Write(rec, req, domainTrip.ErrTripNotFound)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("expected problem+json content type, got %s", rec.Header().Get("Content-Type"))
	}
}

func TestWrite_500_LogsAndSetsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-span")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trips/123", nil).WithContext(ctx)

	secretDBErr := errors.New("pq: password leaked")
	domainerror.Write(rec, req, secretDBErr)
	span.End()

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "password leaked")

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
}
