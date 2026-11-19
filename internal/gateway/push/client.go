package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const kindRequestPosition = "REQUEST_POSITION"

var (
	ErrClientStatus = errors.New("push service 4xx error")
	ErrServerStatus = errors.New("push service 5xx error")
)

type StatusError struct {
	StatusCode int
	Body       string
	Base       error
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("push service returned status %d: %s", e.StatusCode, e.Body)
}

func (e *StatusError) Unwrap() error {
	return e.Base
}

type sendRequest struct {
	RecipientID string            `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	transport  *http.Transport
	tracer     trace.Tracer
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	tr := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(tr),
		},
		transport: tr,
		tracer:    otel.Tracer("trip-service"),
	}
}

func (c *Client) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	ctx, span := c.tracer.Start(ctx, "push.request_position",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("trip.id", tripID.String()),
			attribute.String("driver.id", driverID.String()),
			attribute.String("push.kind", kindRequestPosition),
		),
	)
	defer span.End()

	payload, err := json.Marshal(sendRequest{
		RecipientID: driverID.String(),
		Kind:        kindRequestPosition,
		Data:        map[string]string{"trip_id": tripID.String()},
	})
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "marshal_error"))
		return fmt.Errorf("marshal push request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/pushes", bytes.NewReader(payload))
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "request_build_error"))
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String("error.type", "transport_error"))
		return fmt.Errorf("send push request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	bodyStr := strings.TrimSpace(string(bodyBytes))

	if resp.StatusCode >= http.StatusBadRequest && resp.StatusCode < http.StatusInternalServerError {
		span.SetStatus(codes.Error, "4xx response")
		span.SetAttributes(attribute.String("error.type", "http_4xx"))
		return &StatusError{StatusCode: resp.StatusCode, Body: bodyStr, Base: ErrClientStatus}
	}

	span.SetStatus(codes.Error, "5xx response")
	span.SetAttributes(attribute.String("error.type", "http_5xx"))
	return &StatusError{StatusCode: resp.StatusCode, Body: bodyStr, Base: ErrServerStatus}
}

func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
	c.transport.CloseIdleConnections()
}
