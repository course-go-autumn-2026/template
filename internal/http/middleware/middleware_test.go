package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/whxtelyy/template/internal/generated"
)

func TestRecover(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	handler := Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("test panic") }))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/health", nil))
	var body api.Problem
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusInternalServerError || body.Code != "internal_error" || !strings.Contains(logs.String(), "stack") {
		t.Fatalf("status=%d body=%s logs=%s", w.Code, w.Body.String(), logs.String())
	}
}

func TestRouteUsesTemplate(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/trips/5cb72c04-7650-45c9-a79b-bcdba0631e0c", nil)
	if got := Route(r); got != "/api/v1/trips/{tripId}" {
		t.Fatalf("route = %q", got)
	}
}
