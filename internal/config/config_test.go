package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	const shutdownTimeout = "SHUTDOWN_TIMEOUT"
	const invalid = "invalid"
	values := map[string]string{
		"PUSH_HTTP_URL": "http://localhost:8090", "HTTP_ADDR": " :8080 ", "DATABASE_URL": " postgres://localhost/db ",
		"LOG_LEVEL": " INFO ", shutdownTimeout: "10s",
		"DATABASE_MAX_CONNS": "10", "DATABASE_MIN_CONNS": "0",
		"DATABASE_CONNECT_TIMEOUT": "5s", "DATABASE_QUERY_TIMEOUT": "5s",
		"DATABASE_MAX_CONN_LIFETIME": "1h", "OTEL_SERVICE_NAME": "trip",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "localhost:4317", "OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
		"OTEL_RESOURCE_ATTRIBUTES": "service.namespace=trips", "OTEL_TRACES_SAMPLER": "always_on",
		"HTTP_READ_TIMEOUT": " ", "HTTP_READ_HEADER_TIMEOUT": "", "HTTP_WRITE_TIMEOUT": "", "HTTP_IDLE_TIMEOUT": "",
	}
	for name, value := range values {
		t.Setenv(name, value)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.LogLevel != "info" || cfg.HTTPReadTimeout != 5*time.Second || cfg.HTTPIdleTimeout != time.Minute {
		t.Fatalf("unexpected configuration: %+v", cfg)
	}

	for _, tc := range []struct{ name, value string }{
		{"HTTP_ADDR", "  "},
		{shutdownTimeout, invalid},
		{"HTTP_WRITE_TIMEOUT", "0s"},
		{"DATABASE_MAX_CONNS", "0"},
		{"DATABASE_MIN_CONNS", "11"},
		{"OTEL_EXPORTER_OTLP_PROTOCOL", invalid},
		{"OTEL_TRACES_SAMPLER", invalid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.name, tc.value)
			if _, err := Load(); err == nil || (!strings.Contains(err.Error(), tc.name) && (tc.name != shutdownTimeout || !strings.Contains(err.Error(), "ShutdownTimeout"))) {
				t.Fatalf("expected error mentioning %s, got %v", tc.name, err)
			}
		})
	}
}
