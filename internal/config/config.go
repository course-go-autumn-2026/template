// Package config loads and validates trip service configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Config contains application settings loaded from environment variables.
type Config struct {
	PositionCheckInterval time.Duration `env:"POSITION_CHECK_INTERVAL" envDefault:"30s"`
	PositionStaleAfter    time.Duration `env:"POSITION_STALE_AFTER" envDefault:"60s"`
	PositionWorkers       int           `env:"POSITION_WORKERS" envDefault:"4"`
	PositionQueueSize     int           `env:"POSITION_QUEUE_SIZE" envDefault:"100"`
	PositionScanBatchSize int           `env:"POSITION_SCAN_BATCH_SIZE" envDefault:"100"`
	PushHTTPURL           string        `env:"PUSH_HTTP_URL,required,notEmpty"`
	PushRequestTimeout    time.Duration `env:"PUSH_REQUEST_TIMEOUT" envDefault:"1s"`

	HTTPAddr                string        `env:"HTTP_ADDR,required,notEmpty"`
	HTTPReadTimeout         time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"5s"`
	HTTPReadHeaderTimeout   time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
	HTTPWriteTimeout        time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"10s"`
	HTTPIdleTimeout         time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"1m"`
	DatabaseURL             string        `env:"DATABASE_URL,required,notEmpty"`
	LogLevel                string        `env:"LOG_LEVEL,required,notEmpty"`
	ShutdownTimeout         time.Duration `env:"SHUTDOWN_TIMEOUT,required,notEmpty"`
	DatabaseMaxConns        int32         `env:"DATABASE_MAX_CONNS,required,notEmpty"`
	DatabaseMinConns        int32         `env:"DATABASE_MIN_CONNS,required,notEmpty"`
	DatabaseConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT,required,notEmpty"`
	DatabaseQueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT,required,notEmpty"`
	DatabaseMaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME,required,notEmpty"`
	OTELServiceName         string        `env:"OTEL_SERVICE_NAME,required,notEmpty"`
	OTELExporterEndpoint    string        `env:"OTEL_EXPORTER_OTLP_ENDPOINT,required,notEmpty"`
	OTELExporterProtocol    string        `env:"OTEL_EXPORTER_OTLP_PROTOCOL,required,notEmpty"`
	OTELResourceAttributes  string        `env:"OTEL_RESOURCE_ATTRIBUTES,required,notEmpty"`
	OTELTracesSampler       string        `env:"OTEL_TRACES_SAMPLER,required,notEmpty"`
}

// Load reads and validates the required service configuration from the environment.
func Load() (Config, error) {
	// Preserve the previous behavior: ignore surrounding whitespace, and treat
	// whitespace-only values as absent (including for optional defaults).
	environment := make(map[string]string)
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if value = strings.TrimSpace(value); value != "" {
			environment[name] = value
		}
	}

	cfg, err := env.ParseAsWithOptions[Config](env.Options{Environment: environment})
	if err != nil {
		return Config{}, fmt.Errorf("load environment configuration: %w", err)
	}

	for _, timeout := range []struct {
		name  string
		value time.Duration
	}{
		{"POSITION_CHECK_INTERVAL", cfg.PositionCheckInterval},
		{"POSITION_STALE_AFTER", cfg.PositionStaleAfter},
		{"PUSH_REQUEST_TIMEOUT", cfg.PushRequestTimeout},
		{"HTTP_READ_TIMEOUT", cfg.HTTPReadTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", cfg.HTTPReadHeaderTimeout},
		{"HTTP_WRITE_TIMEOUT", cfg.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", cfg.HTTPIdleTimeout},
		{"SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout},
		{"DATABASE_CONNECT_TIMEOUT", cfg.DatabaseConnectTimeout},
		{"DATABASE_QUERY_TIMEOUT", cfg.DatabaseQueryTimeout},
		{"DATABASE_MAX_CONN_LIFETIME", cfg.DatabaseMaxConnLifetime},
	} {
		if timeout.value <= 0 {
			return Config{}, fmt.Errorf("%s must be greater than zero", timeout.name)
		}
	}

	if cfg.PositionWorkers < 1 || cfg.PositionQueueSize < 1 || cfg.PositionScanBatchSize < 1 {
		return Config{}, fmt.Errorf("position workers, queue size and batch size must be positive")
	}
	cfg.LogLevel = strings.ToLower(cfg.LogLevel)
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error; got %q", cfg.LogLevel)
	}

	if cfg.DatabaseMaxConns < 1 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if cfg.DatabaseMinConns < 0 || cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must be between zero and DATABASE_MAX_CONNS")
	}

	switch cfg.OTELExporterProtocol {
	case "http/protobuf", "grpc":
	default:
		return Config{}, fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be http/protobuf or grpc; got %q", cfg.OTELExporterProtocol)
	}
	switch cfg.OTELTracesSampler {
	case "always_on", "always_off", "parentbased_always_on", "parentbased_always_off":
	default:
		return Config{}, fmt.Errorf("OTEL_TRACES_SAMPLER must be one of always_on, always_off, parentbased_always_on, parentbased_always_off; got %q", cfg.OTELTracesSampler)
	}

	return cfg, nil
}
