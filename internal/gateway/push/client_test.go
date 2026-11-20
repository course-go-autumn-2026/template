package push

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRequest(t *testing.T) {
	trip, driver := uuid.New(), uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/push" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			RecipientID uuid.UUID         `json:"recipient_id"`
			Kind        string            `json:"kind"`
			Data        map[string]string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.RecipientID != driver || body.Kind != "REQUEST_POSITION" || body.Data["trip_id"] != trip.String() {
			t.Errorf("unexpected body: %+v", body)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	client := New(server.URL, time.Second)
	defer client.Close()
	if err := client.Request(context.Background(), trip, driver); err != nil {
		t.Fatal(err)
	}
}

func TestRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer server.Close()
	client := New(server.URL, time.Second)
	defer client.Close()
	if err := client.Request(context.Background(), uuid.New(), uuid.New()); err == nil {
		t.Fatal("expected HTTP error")
	}
}
