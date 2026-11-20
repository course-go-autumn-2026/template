package config

import (
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
	validConfig := func() Config {
		return Config{
			HTTPAddr:        ":8080",
			LogLevel:        "info",
			ShutdownTimeout: 10 * time.Second,
			IdempotencyTTL:  24 * time.Hour,
			// #nosec G101 -- default local dev postgres connection string for test environment
			DatabaseURL:             "postgres://tripgo:tripgo@localhost:23032/tripgo?sslmode=disable",
			DatabaseMaxConns:        10,
			DatabaseMinConns:        2,
			DatabaseMaxConnLifetime: 30 * time.Minute,
			DatabaseConnectTimeout:  5 * time.Second,
			DatabaseQueryTimeout:    3 * time.Second,
			PositionCheckInterval:   30 * time.Second,
			PositionStaleAfter:      60 * time.Second,
			PositionWorkers:         4,
			PositionQueueSize:       100,
			PositionScanBatchSize:   100,
			PushHTTPURL:             "http://localhost:23809",
			PushRequestTimeout:      1 * time.Second,
		}
	}

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		if err := cfg.Validate(); err != nil {
			t.Fatalf("unexpected validation error: %v", err)
		}
	})

	t.Run("missing http addr", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.HTTPAddr = ""
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for empty HTTPAddr, got nil")
		}
	})

	t.Run("invalid log level", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.LogLevel = "verbose"
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for invalid LogLevel, got nil")
		}
	})

	t.Run("min conns greater than max conns", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.DatabaseMinConns = 15
		cfg.DatabaseMaxConns = 10
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for DatabaseMinConns > DatabaseMaxConns, got nil")
		}
	})

	t.Run("negative database min conns", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.DatabaseMinConns = -1
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for negative DatabaseMinConns, got nil")
		}
	})

	t.Run("non positive duration", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.ShutdownTimeout = 0
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for zero ShutdownTimeout, got nil")
		}
	})

	t.Run("invalid database url", func(t *testing.T) {
		t.Parallel()
		cfg := validConfig()
		cfg.DatabaseURL = "not-a-valid-url"
		if err := cfg.Validate(); err == nil {
			t.Fatalf("expected error for invalid DatabaseURL, got nil")
		}
	})
}

func TestConfig_Load(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config from .env: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}
