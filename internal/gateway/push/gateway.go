package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	requestPositionKind = "REQUEST_POSITION"
	pushEndpoint        = "/api/v1/push"
)

var ErrTransport = errors.New("push service transport error")

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("push service status %d: %s", e.StatusCode, e.Body)
}

type Gateway struct {
	endpoint string
	client   *http.Client
	tracer   trace.Tracer
}

func New(baseURL string, timeout time.Duration) *Gateway {
	return NewWithClient(baseURL, &http.Client{
		Timeout: timeout,
	})
}

func NewWithClient(baseURL string, client *http.Client) *Gateway {
	return &Gateway{
		endpoint: strings.TrimRight(baseURL, "/") + pushEndpoint,
		client:   client,
		tracer:   otel.GetTracerProvider().Tracer("trip-service"),
	}
}

type pushPayload struct {
	RequestID   string            `json:"request_id"`
	RecipientID uuid.UUID         `json:"recipient_id"`
	Kind        string            `json:"kind"`
	Data        map[string]string `json:"data"`
}

func (g *Gateway) RequestPosition(ctx context.Context, driverID, tripID uuid.UUID) error {
	ctx, span := g.tracer.Start(ctx, "push_service.request_position")
	defer span.End()

	payload := pushPayload{
		RequestID:   uuid.NewString(),
		RecipientID: driverID,
		Kind:        requestPositionKind,
		Data: map[string]string{
			"trip_id": tripID.String(),
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal push payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		transportErr := fmt.Errorf("%w: %v", ErrTransport, err)
		span.RecordError(transportErr)
		span.SetStatus(codes.Error, transportErr.Error())
		return transportErr
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusAccepted {
		limitedBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		statusErr := &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Body:       string(limitedBody),
		}
		span.RecordError(statusErr)
		span.SetStatus(codes.Error, statusErr.Error())
		return statusErr
	}

	span.SetStatus(codes.Ok, "accepted")
	return nil
}
