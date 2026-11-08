package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupProviders(t *testing.T, logger *slog.Logger) (*Providers, <-chan struct{}) {
	t.Helper()
	metricExports := make(chan struct{}, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if r.URL.Path == "/v1/metrics" {
			select {
			case metricExports <- struct{}{}:
			default:
			}
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	t.Setenv("OTEL_SERVICE_NAME", "test-trip-service")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", server.URL)
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	previous := otel.GetErrorHandler()
	t.Cleanup(func() { otel.SetErrorHandler(previous) })
	providers, err := New(context.Background(), logger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := providers.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	return providers, metricExports
}

func TestSamplerEnvironmentAndVersion(t *testing.T) {
	for _, tc := range []struct {
		sampler, arg string
		recording    bool
	}{
		{"always_on", "", true},
		{"always_off", "", false},
		{"traceidratio", "0", false},
		{"parentbased_traceidratio", "1", true},
		{"", "", true},
	} {
		t.Run(tc.sampler, func(t *testing.T) {
			t.Setenv("OTEL_TRACES_SAMPLER", tc.sampler)
			t.Setenv("OTEL_TRACES_SAMPLER_ARG", tc.arg)
			p, _ := setupProviders(t, slog.New(slog.NewJSONHandler(io.Discard, nil)))
			recorder := tracetest.NewSpanRecorder()
			p.Trace.RegisterSpanProcessor(recorder)
			_, span := p.Trace.Tracer("test").Start(context.Background(), "test")
			if span.IsRecording() != tc.recording {
				t.Fatalf("IsRecording=%v", span.IsRecording())
			}
			span.End()
			if tc.recording {
				spans := recorder.Ended()
				if len(spans) != 1 {
					t.Fatalf("spans=%v", spans)
				}
				version, _ := spans[0].Resource().Set().Value("service.version")
				if version.AsString() != Version {
					t.Fatalf("version=%s, want %s", version.AsString(), Version)
				}
			}
		})
	}
}

func TestMetricExportIntervalEnvironment(t *testing.T) {
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "20")
	p, exports := setupProviders(t, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	p.RecordCreated(context.Background())
	select {
	case <-exports:
	case <-time.After(3 * time.Second):
		t.Fatal("SDK did not use OTEL_METRIC_EXPORT_INTERVAL=20ms")
	}
}

func TestErrorHandlerWritesJSONWarning(t *testing.T) {
	var logs bytes.Buffer
	_, _ = setupProviders(t, slog.New(slog.NewJSONHandler(&logs, nil)))
	otel.Handle(errors.New("test export failure"))
	var entry struct{ Level, Msg, Error string }
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Level != "WARN" || entry.Error != "test export failure" {
		t.Fatalf("log=%s", logs.String())
	}
}
