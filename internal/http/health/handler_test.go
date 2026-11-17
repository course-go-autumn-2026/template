package health_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/freyliee/template/internal/http/health"
)

func TestHealth(t *testing.T) {
	ctrl := gomock.NewController(t)
	db := NewMockdatabase(ctrl)
	handler := health.New(db, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	w := httptest.NewRecorder()
	handler.Health(w, httptest.NewRequest("GET", "/health", nil))

	if w.Code != 200 || !strings.Contains(w.Body.String(), "ok") {
		t.Fatalf("unexpected health response: %s", w.Body.String())
	}
}

func TestReady(t *testing.T) {
	for _, failure := range []error{nil, errors.New("unavailable")} {
		name := "ready"

		if failure != nil {
			name = "unavailable"
		}

		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			db := NewMockdatabase(ctrl)
			db.EXPECT().Ping(gomock.Any()).DoAndReturn(func(ctx context.Context) error {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > time.Second {
					t.Fatal("database ping has no bounded timeout")
				}

				return failure
			})

			handler := health.New(db, time.Second, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			w := httptest.NewRecorder()
			handler.Ready(w, httptest.NewRequest("GET", "/ready", nil))

			status := 200

			if failure != nil {
				status = 503
			}

			if w.Code != status || (failure == nil && !strings.Contains(w.Body.String(), "ok") || failure != nil && !strings.Contains(w.Body.String(), "unavailable")) {
				t.Fatalf("unexpected ready response: %d %s", w.Code, w.Body.String())
			}
		})
	}
}
