package config

import (
	"os"
	"testing"
)

var testEnv = map[string]string{
	"HTTP_ADDR":                  ":8080",
	"DATABASE_URL":               "postgres://test:test@localhost:5432/test?sslmode=disable",
	"LOG_LEVEL":                  "info",
	"SHUTDOWN_TIMEOUT":           "10s",
	"DATABASE_MAX_CONNS":         "10",
	"DATABASE_MIN_CONNS":         "2",
	"DATABASE_CONNECT_TIMEOUT":   "5s",
	"DATABASE_QUERY_TIMEOUT":     "5s",
	"DATABASE_MAX_CONN_LIFETIME": "1h",
	"HTTP_READ_TIMEOUT":          "5s",
	"HTTP_WRITE_TIMEOUT":         "10s",
	"HTTP_IDLE_TIMEOUT":          "60s",
	"READY_TIMEOUT":              "5s",
}

func setValidEnv(t *testing.T) {
	t.Helper()

	for key, value := range testEnv {
		t.Setenv(key, value)
	}
}

func TestLoadValidConfig(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}

	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, ":8080")
	}
	if cfg.DatabaseMaxConns != 10 {
		t.Errorf("DatabaseMaxConns = %d, want 10", cfg.DatabaseMaxConns)
	}
	if cfg.DatabaseMinConns != 2 {
		t.Errorf("DatabaseMinConns = %d, want 2", cfg.DatabaseMinConns)
	}
}

func TestLoadMissingDatabaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error when DATABASE_URL is empty")
	}
}

func TestLoadInvalidLogLevel(t *testing.T) {
	setValidEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for invalid LOG_LEVEL")
	}
}

func TestLoadMinConnsGreaterThanMax(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_MAX_CONNS", "2")
	t.Setenv("DATABASE_MIN_CONNS", "5")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error when min connections exceed max connections")
	}
}

func TestLoadInvalidDuration(t *testing.T) {
	setValidEnv(t)
	t.Setenv("SHUTDOWN_TIMEOUT", "later")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error for invalid duration")
	}
}

func TestLoadMissingRequiredVariable(t *testing.T) {
	setValidEnv(t)

	oldValue, existed := os.LookupEnv("HTTP_ADDR")
	if existed {
		t.Cleanup(func() {
			_ = os.Setenv("HTTP_ADDR", oldValue)
		})
	}

	t.Setenv("HTTP_ADDR", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected an error when HTTP_ADDR is empty")
	}
}
