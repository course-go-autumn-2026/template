package push_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/George1487/MalyginGeorge/internal/gateway/push"
)

type roundTripperFunc func(req *http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGateway_RequestPosition(t *testing.T) {
	t.Parallel()

	driverID := uuid.New()
	tripID := uuid.New()

	t.Run("success 202 accepted", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/v1/push", r.URL.Path)
				require.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var req struct {
					RequestID   string            `json:"request_id"`
					RecipientID uuid.UUID         `json:"recipient_id"`
					Kind        string            `json:"kind"`
					Data        map[string]string `json:"data"`
				}
				err = json.Unmarshal(body, &req)
				require.NoError(t, err)
				require.NotEmpty(t, req.RequestID)
				require.Equal(t, driverID, req.RecipientID)
				require.Equal(t, "REQUEST_POSITION", req.Kind)
				require.Equal(t, tripID.String(), req.Data["trip_id"])

				return &http.Response{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(bytes.NewBufferString(`{"message_id":"msg-123","accepted_at":"2026-08-24T12:00:00Z"}`)),
					Header:     make(http.Header),
				}, nil
			}),
		}

		gw := push.NewWithClient("http://localhost:8090", client)
		err := gw.RequestPosition(context.Background(), driverID, tripID)
		require.NoError(t, err)
	})

	t.Run("server returns 400 bad request", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewBufferString("bad request format")),
					Header:     make(http.Header),
				}, nil
			}),
		}

		gw := push.NewWithClient("http://localhost:8090", client)
		err := gw.RequestPosition(context.Background(), driverID, tripID)
		require.Error(t, err)

		var httpErr *push.HTTPStatusError
		require.True(t, errors.As(err, &httpErr))
		require.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
		require.Equal(t, "bad request format", httpErr.Body)
	})

	t.Run("server returns 503 unavailable", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Body:       io.NopCloser(bytes.NewBufferString("service unavailable")),
					Header:     make(http.Header),
				}, nil
			}),
		}

		gw := push.NewWithClient("http://localhost:8090", client)
		err := gw.RequestPosition(context.Background(), driverID, tripID)
		require.Error(t, err)

		var httpErr *push.HTTPStatusError
		require.True(t, errors.As(err, &httpErr))
		require.Equal(t, http.StatusServiceUnavailable, httpErr.StatusCode)
		require.Equal(t, "service unavailable", httpErr.Body)
	})

	t.Run("transport error on network failure", func(t *testing.T) {
		t.Parallel()

		client := &http.Client{
			Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
				return nil, errors.New("network unreachable")
			}),
		}

		gw := push.NewWithClient("http://localhost:8090", client)
		err := gw.RequestPosition(context.Background(), driverID, tripID)
		require.Error(t, err)
		require.True(t, errors.Is(err, push.ErrTransport))
	})
}
