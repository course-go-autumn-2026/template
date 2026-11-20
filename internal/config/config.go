package config

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

var validate = validator.New()

type Config struct {
	HTTPAddr        string        `env:"HTTP_ADDR,required,notEmpty" validate:"required"`
	LogLevel        string        `env:"LOG_LEVEL,required,notEmpty" validate:"required,oneof=debug info warn error DEBUG INFO WARN ERROR"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT,required" validate:"gt=0"`
	IdempotencyTTL  time.Duration `env:"IDEMPOTENCY_TTL" envDefault:"24h" validate:"gt=0"`

	DatabaseURL             string        `env:"DATABASE_URL,required,notEmpty" validate:"required,url"`
	DatabaseMaxConns        int32         `env:"DATABASE_MAX_CONNS,required" validate:"gt=0"`
	DatabaseMinConns        int32         `env:"DATABASE_MIN_CONNS,required" validate:"gte=0,ltefield=DatabaseMaxConns"`
	DatabaseMaxConnLifetime time.Duration `env:"DATABASE_MAX_CONN_LIFETIME,required" validate:"gt=0"`
	DatabaseConnectTimeout  time.Duration `env:"DATABASE_CONNECT_TIMEOUT,required" validate:"gt=0"`
	DatabaseQueryTimeout    time.Duration `env:"DATABASE_QUERY_TIMEOUT,required" validate:"gt=0"`

	OTelServiceName          string `env:"OTEL_SERVICE_NAME" envDefault:"trip-service"`
	OTelExporterOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
	OTelExporterOTLPProtocol string `env:"OTEL_EXPORTER_OTLP_PROTOCOL" envDefault:"http/protobuf"`
	OTelResourceAttributes   string `env:"OTEL_RESOURCE_ATTRIBUTES"`
	OTelTracesSampler        string `env:"OTEL_TRACES_SAMPLER" envDefault:"parentbased_always_on"`

	PositionCheckInterval time.Duration `env:"POSITION_CHECK_INTERVAL" envDefault:"30s" validate:"gt=0"`
	PositionStaleAfter    time.Duration `env:"POSITION_STALE_AFTER" envDefault:"60s" validate:"gt=0"`
	PositionWorkers       int           `env:"POSITION_WORKERS" envDefault:"4" validate:"gt=0"`
	PositionQueueSize     int           `env:"POSITION_QUEUE_SIZE" envDefault:"100" validate:"gt=0"`
	PositionScanBatchSize int           `env:"POSITION_SCAN_BATCH_SIZE" envDefault:"100" validate:"gt=0"`
	PushHTTPURL           string        `env:"PUSH_HTTP_URL" validate:"omitempty,url"`
	PushRequestTimeout    time.Duration `env:"PUSH_REQUEST_TIMEOUT" envDefault:"1s" validate:"gt=0"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if err := validate.Struct(c); err != nil {
		return fmt.Errorf("validate struct: %w", err)
	}
	return nil
}

func (c *Config) ParsedLogLevel() slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(c.LogLevel)); err != nil {
		return slog.LevelInfo
	}
	return level
}
