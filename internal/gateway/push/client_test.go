package push_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/xotapy-main/courseGo/internal/gateway/push"
)

func TestClient_RequestPosition(t *testing.T) {
	t.Parallel()

	driverID := uuid.New()
	tripID := uuid.New()

	t.Run("200 ok", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/api/v1/pushes", r.URL.Path)

			var body struct {
				RecipientID string            `json:"recipient_id"`
				Kind        string            `json:"kind"`
				Data        map[string]string `json:"data"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, driverID.String(), body.RecipientID)
			require.Equal(t, "REQUEST_POSITION", body.Kind)
			require.Equal(t, tripID.String(), body.Data["trip_id"])

			w.WriteHeader(http.StatusAccepted)
		}))
		defer srv.Close()

		client := push.NewClient(srv.URL, time.Second)
		defer client.Close()

		require.NoError(t, client.RequestPosition(context.Background(), driverID, tripID))
	})

	t.Run("4xx client error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad payload"))
		}))
		defer srv.Close()

		client := push.NewClient(srv.URL, time.Second)
		defer client.Close()

		err := client.RequestPosition(context.Background(), driverID, tripID)
		require.ErrorIs(t, err, push.ErrClientStatus)
	})

	t.Run("5xx server error", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("unavailable"))
		}))
		defer srv.Close()

		client := push.NewClient(srv.URL, time.Second)
		defer client.Close()

		err := client.RequestPosition(context.Background(), driverID, tripID)
		require.ErrorIs(t, err, push.ErrServerStatus)
	})

	t.Run("timeout transport error", func(t *testing.T) {
		t.Parallel()
		block := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			<-block
			w.WriteHeader(http.StatusOK)
		}))
		defer func() {
			close(block)
			srv.Close()
		}()

		client := push.NewClient(srv.URL, 50*time.Millisecond)
		defer client.Close()

		err := client.RequestPosition(context.Background(), driverID, tripID)
		require.Error(t, err)
		require.NotErrorIs(t, err, push.ErrClientStatus)
		require.NotErrorIs(t, err, push.ErrServerStatus)
	})
}
