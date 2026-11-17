//go:generate go tool mockgen -destination mocks_test.go -package push net/http RoundTripper
package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type HTTPError struct{ StatusCode int }

func (e *HTTPError) Error() string { return fmt.Sprintf("push service returned HTTP %d", e.StatusCode) }

type TransportError struct{ Err error }

func (e *TransportError) Error() string { return "push transport: " + e.Err.Error() }

func (e *TransportError) Unwrap() error { return e.Err }

type Client struct {
	client     *http.Client
	url        string
	tracer     trace.Tracer
	propagator propagation.TextMapPropagator
}

func New(url string, timeout time.Duration, tracer trace.Tracer, propagator propagation.TextMapPropagator) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: timeout, KeepAlive: timeout}).DialContext
	transport.TLSHandshakeTimeout = timeout
	transport.ResponseHeaderTimeout = timeout
	transport.IdleConnTimeout = timeout

	return &Client{
		client: &http.Client{
			Transport:     transport,
			Timeout:       timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		},
		url:        strings.TrimRight(url, "/") + "/api/v1/push",
		tracer:     tracer,
		propagator: propagator,
	}
}

func (c *Client) RequestPosition(ctx context.Context, tripID, driverID uuid.UUID) (result error) {
	ctx, span := c.tracer.Start(ctx, "POST /api/v1/push", trace.WithSpanKind(trace.SpanKindClient))
	defer func() {
		if result != nil {
			span.RecordError(result)
			span.SetStatus(codes.Error, "push request failed")
			span.SetAttributes(attribute.String("error.type", fmt.Sprintf("%T", result)))
		}

		span.End()
	}()

	payload := struct {
		RequestID   string            `json:"request_id"`
		RecipientID uuid.UUID         `json:"recipient_id"`
		Kind        string            `json:"kind"`
		Data        map[string]string `json:"data"`
	}{
		uuid.NewString(),
		driverID,
		"REQUEST_POSITION",
		map[string]string{"trip_id": tripID.String()},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode push: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	c.propagator.Inject(ctx, propagation.HeaderCarrier(request.Header))
	response, err := c.client.Do(request)
	if err != nil {
		return &TransportError{Err: err}
	}

	defer response.Body.Close()

	span.SetAttributes(attribute.Int("http.response.status_code", response.StatusCode))

	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10)); err != nil {
		return &TransportError{Err: fmt.Errorf("read push response: %w", err)}
	}

	if response.StatusCode != http.StatusAccepted {
		return &HTTPError{StatusCode: response.StatusCode}
	}

	return nil
}

func (c *Client) Close() { c.client.CloseIdleConnections() }
