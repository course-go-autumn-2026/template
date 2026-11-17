package push

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true

	return nil
}

func TestRequestPosition(t *testing.T) {
	for _, status := range []int{202, 400, 429, 503, 302} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			transport := NewMockRoundTripper(gomock.NewController(t))
			c := New(
				"http://push.test",
				time.Second,
				noop.NewTracerProvider().Tracer("test"),
				propagation.TraceContext{},
			)
			defer c.Close()

			c.client.Transport = transport
			body := &trackedBody{Reader: strings.NewReader(`{}`)}
			tripID, driverID := uuid.New(), uuid.New()
			parent := trace.NewSpanContext(trace.SpanContextConfig{TraceID: trace.TraceID{1}, SpanID: trace.SpanID{2}, TraceFlags: trace.FlagsSampled})
			ctx := trace.ContextWithSpanContext(context.Background(), parent)
			transport.EXPECT().RoundTrip(gomock.Any()).DoAndReturn(func(request *http.Request) (*http.Response, error) {
				if request.Method != http.MethodPost || request.URL.Path != "/api/v1/push" || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("traceparent") == "" {
					t.Errorf("invalid push request: %+v", request)
				}

				var payload struct {
					RequestID   string            `json:"request_id"`
					RecipientID uuid.UUID         `json:"recipient_id"`
					Kind        string            `json:"kind"`
					Data        map[string]string `json:"data"`
				}

				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Error(err)
				}

				if payload.RequestID == "" || payload.RecipientID != driverID || payload.Kind != "REQUEST_POSITION" || payload.Data["trip_id"] != tripID.String() {
					t.Errorf("invalid push: %+v", payload)
				}

				return &http.Response{StatusCode: status, Body: body, Header: make(http.Header)}, nil
			})
			err := c.RequestPosition(ctx, tripID, driverID)

			if status == 202 && err != nil {
				t.Fatal(err)
			}

			if status != 202 {
				var responseError *HTTPError

				if !errors.As(err, &responseError) || responseError.StatusCode != status {
					t.Fatalf("status error: %v", err)
				}
			}

			if !body.closed {
				t.Fatal("response body not closed")
			}
		})
	}
}

func TestTransportFailure(t *testing.T) {
	transport := NewMockRoundTripper(gomock.NewController(t))
	c := New(
		"http://push.test",
		time.Second,
		noop.NewTracerProvider().Tracer("test"),
		propagation.TraceContext{},
	)
	defer c.Close()

	c.client.Transport = transport
	failure := errors.New("network unavailable")
	transport.EXPECT().RoundTrip(gomock.Any()).Return(nil, failure)
	err := c.RequestPosition(context.Background(), uuid.New(), uuid.New())

	var transportError *TransportError

	if !errors.As(err, &transportError) || !errors.Is(err, failure) {
		t.Fatalf("transport error: %v", err)
	}
}
