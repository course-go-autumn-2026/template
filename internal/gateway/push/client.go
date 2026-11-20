package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client struct {
	client *http.Client
	url    string
}

func New(url string, timeout time.Duration) *Client {
	return &Client{client: &http.Client{Timeout: timeout, Transport: otelhttp.NewTransport(http.DefaultTransport.(*http.Transport).Clone())}, url: strings.TrimRight(url, "/") + "/api/v1/push"}
}

func (c *Client) Request(ctx context.Context, trip, driver uuid.UUID) error {
	body, err := json.Marshal(map[string]any{"request_id": uuid.NewString(), "recipient_id": driver, "kind": "REQUEST_POSITION", "data": map[string]string{"trip_id": trip.String()}})
	if err != nil {
		return fmt.Errorf("encode push: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("push transport: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("push HTTP status: %d", resp.StatusCode)
	}
	return nil
}
func (c *Client) Close() { c.client.CloseIdleConnections() }
