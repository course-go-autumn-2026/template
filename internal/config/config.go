package config

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	OTLPEndpoint         string
	MetricExportInterval time.Duration
	ExportTimeout        time.Duration
	OTLPProtocol         string
	TracesSampler        string
	SamplerArg           float64
	ServiceVersion       string

	PositionCheckInterval time.Duration
	PositionStaleAfter    time.Duration
	PositionWorkers       int
	PositionQueueSize     int
	PushHTTPURL           string
	PushRequestTimeout    time.Duration

	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration

	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
}

func Load() (Config, error) {
	var cfg Config

	cfg.HTTPAddr = os.Getenv("HTTP_ADDR")
	if cfg.HTTPAddr == "" {
		return cfg, fmt.Errorf("HTTP_ADDR is required")
	}

	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return cfg, fmt.Errorf(
			"LOG_LEVEL must be debug, info, warn or error",
		)
	}

	value := os.Getenv("SHUTDOWN_TIMEOUT")
	timeout, err := time.ParseDuration(value)
	if err != nil || timeout <= 0 {
		return cfg, fmt.Errorf(
			"SHUTDOWN_TIMEOUT must be a positive duration, for example 10s",
		)
	}
	cfg.ShutdownTimeout = timeout

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return cfg, fmt.Errorf("DATABASE_URL is required")
	}

	cfg.DatabaseMaxConns, err = readInt32("DATABASE_MAX_CONNS")
	if err != nil {
		return cfg, err
	}
	if cfg.DatabaseMaxConns <= 0 {
		return cfg, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}

	cfg.DatabaseMinConns, err = readInt32("DATABASE_MIN_CONNS")
	if err != nil {
		return cfg, err
	}
	if cfg.DatabaseMinConns < 0 ||
		cfg.DatabaseMinConns > cfg.DatabaseMaxConns {
		return cfg, fmt.Errorf(
			"DATABASE_MIN_CONNS must be between 0 and DATABASE_MAX_CONNS",
		)
	}

	cfg.DatabaseConnectTimeout, err = readDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return cfg, err
	}

	cfg.DatabaseQueryTimeout, err = readDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return cfg, err
	}

	cfg.DatabaseMaxConnLifetime, err = readDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return cfg, err
	}

	for _, setting := range []struct {
		name        string
		destination *time.Duration
	}{
		{"HTTP_READ_TIMEOUT", &cfg.HTTPReadTimeout}, {"HTTP_READ_HEADER_TIMEOUT", &cfg.HTTPReadHeaderTimeout}, {"HTTP_WRITE_TIMEOUT", &cfg.HTTPWriteTimeout}, {"HTTP_IDLE_TIMEOUT", &cfg.HTTPIdleTimeout},
	} {
		value, err := readDuration(setting.name)
		if err != nil {
			return cfg, err
		}

		*setting.destination = value
	}

	if err := loadPositions(&cfg); err != nil {
		return cfg, err
	}

	if err := loadTelemetry(&cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func readInt32(name string) (int32, error) {
	value, err := strconv.ParseInt(os.Getenv(name), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be an int32: %w", name, err)
	}

	return int32(value), nil
}

func readDuration(name string) (time.Duration, error) {
	value, err := time.ParseDuration(os.Getenv(name))
	if err != nil {
		return 0, fmt.Errorf("%s must be a duration: %w", name, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}

	return value, nil
}

func loadTelemetry(cfg *Config) error {
	cfg.OTLPEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	if cfg.OTLPEndpoint == "" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required")
	}

	endpoint, err := url.Parse(cfg.OTLPEndpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT must be an http(s) URL without credentials, query or fragment")
	}

	for _, setting := range []struct {
		name        string
		destination *time.Duration
	}{
		{"OTEL_METRIC_EXPORT_INTERVAL", &cfg.MetricExportInterval},
		{"OTEL_EXPORTER_OTLP_TIMEOUT", &cfg.ExportTimeout},
	} {
		milliseconds, err := strconv.ParseInt(os.Getenv(setting.name), 10, 64)
		if err != nil || milliseconds <= 0 || milliseconds > math.MaxInt64/int64(time.Millisecond) {
			return fmt.Errorf("%s must be positive milliseconds", setting.name)
		}

		*setting.destination = time.Duration(milliseconds) * time.Millisecond
	}

	cfg.OTLPProtocol = os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")

	if cfg.OTLPProtocol != "grpc" && cfg.OTLPProtocol != "http/protobuf" {
		return fmt.Errorf("OTEL_EXPORTER_OTLP_PROTOCOL must be grpc or http/protobuf")
	}

	cfg.TracesSampler = os.Getenv("OTEL_TRACES_SAMPLER")

	switch cfg.TracesSampler {
	case "always_on", "always_off", "traceidratio", "parentbased_always_on", "parentbased_always_off", "parentbased_traceidratio":
	default:
		return fmt.Errorf("invalid OTEL_TRACES_SAMPLER")
	}

	cfg.SamplerArg = 1

	if value := os.Getenv("OTEL_TRACES_SAMPLER_ARG"); value != "" {
		ratio, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(ratio) || ratio < 0 || ratio > 1 {
			return fmt.Errorf("OTEL_TRACES_SAMPLER_ARG must be between 0 and 1")
		}

		cfg.SamplerArg = ratio
	}

	cfg.ServiceVersion = os.Getenv("SERVICE_VERSION")

	if cfg.ServiceVersion == "" {
		cfg.ServiceVersion = "dev"
	}

	return nil
}

func loadPositions(cfg *Config) error {
	for _, setting := range []struct {
		name, fallback string
		destination    *time.Duration
	}{
		{"POSITION_CHECK_INTERVAL", "30s", &cfg.PositionCheckInterval},
		{"POSITION_STALE_AFTER", "60s", &cfg.PositionStaleAfter},
		{"PUSH_REQUEST_TIMEOUT", "1s", &cfg.PushRequestTimeout},
	} {
		value, exists := os.LookupEnv(setting.name)

		if !exists {
			value = setting.fallback
		}

		duration, err := time.ParseDuration(value)

		if err != nil || duration <= 0 {
			return fmt.Errorf("%s must be a positive duration", setting.name)
		}

		*setting.destination = duration
	}

	for _, setting := range []struct {
		name, fallback string
		destination    *int
	}{
		{"POSITION_WORKERS", "4", &cfg.PositionWorkers}, {"POSITION_QUEUE_SIZE", "100", &cfg.PositionQueueSize},
	} {
		value, exists := os.LookupEnv(setting.name)

		if !exists {
			value = setting.fallback
		}

		number, err := strconv.Atoi(value)

		if err != nil || number <= 0 {
			return fmt.Errorf("%s must be a positive integer", setting.name)
		}

		*setting.destination = number
	}

	cfg.PushHTTPURL = os.Getenv("PUSH_HTTP_URL")
	endpoint, err := url.Parse(cfg.PushHTTPURL)

	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return fmt.Errorf("PUSH_HTTP_URL is required and must be an http(s) URL without credentials, query or fragment")
	}

	return nil
}
