package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr              string
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	ShutdownTimeout       time.Duration
	LogLevel              string
	ParsedLogLevel        slog.Level

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration

	OtelServiceName          string
	OtelExporterOtlpEndpoint string
	OtelExporterOtlpProtocol string
	OtelResourceAttributes   string
	OtelTracesSampler        string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if strings.TrimSpace(dbURL) == "" {
		return nil, fmt.Errorf("переменная DATABASE_URL обязательна, но не задана")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	readTimeout, err := parseDurationEnv("HTTP_READ_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}

	readHeaderTimeout, err := parseDurationEnv("HTTP_READ_HEADER_TIMEOUT", "2s")
	if err != nil {
		return nil, err
	}

	writeTimeout, err := parseDurationEnv("HTTP_WRITE_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}

	idleTimeout, err := parseDurationEnv("HTTP_IDLE_TIMEOUT", "30s")
	if err != nil {
		return nil, err
	}

	shutdownTimeout, err := parseDurationEnv("SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return nil, err
	}

	logLevelStr := os.Getenv("LOG_LEVEL")
	if logLevelStr == "" {
		logLevelStr = "info"
	}
	parsedLevel, err := parseLogLevel(logLevelStr)
	if err != nil {
		return nil, err
	}

	maxConns, err := parsePositiveIntEnv("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}

	minConns, err := parseNonNegativeIntEnv("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}

	if minConns > maxConns {
		return nil, fmt.Errorf("некорректная конфигурация пула: DATABASE_MIN_CONNS (%d) не может превышать DATABASE_MAX_CONNS (%d)", minConns, maxConns)
	}

	maxConnLifetime, err := parseDurationEnv("DATABASE_MAX_CONN_LIFETIME", "30m")
	if err != nil {
		return nil, err
	}

	connectTimeout, err := parseDurationEnv("DATABASE_CONNECT_TIMEOUT", "5s")
	if err != nil {
		return nil, err
	}

	queryTimeout, err := parseDurationEnv("DATABASE_QUERY_TIMEOUT", "3s")
	if err != nil {
		return nil, err
	}

	otelService := os.Getenv("OTEL_SERVICE_NAME")
	if otelService == "" {
		otelService = "trip-service"
	}

	protocol := os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	if protocol == "" {
		protocol = "http/protobuf"
	}
	if protocol != "http/protobuf" {
		return nil, fmt.Errorf("неподдерживаемый OTEL_EXPORTER_OTLP_PROTOCOL: %q (поддерживается http/protobuf)", protocol)
	}

	sampler := os.Getenv("OTEL_TRACES_SAMPLER")
	if sampler == "" {
		sampler = "parentbased_always_on"
	}
	switch strings.ToLower(sampler) {
	case "always_on", "always_off", "parentbased_always_on", "parentbased_always_off":
	default:
		return nil, fmt.Errorf("неподдерживаемый OTEL_TRACES_SAMPLER: %q", sampler)
	}

	return &Config{
		HTTPAddr:                 httpAddr,
		HTTPReadTimeout:          readTimeout,
		HTTPReadHeaderTimeout:    readHeaderTimeout,
		HTTPWriteTimeout:         writeTimeout,
		HTTPIdleTimeout:          idleTimeout,
		ShutdownTimeout:          shutdownTimeout,
		LogLevel:                 logLevelStr,
		ParsedLogLevel:           parsedLevel,
		DatabaseURL:              dbURL,
		DatabaseMaxConns:         maxConns,
		DatabaseMinConns:         minConns,
		DatabaseMaxConnLifetime:  maxConnLifetime,
		DatabaseConnectTimeout:   connectTimeout,
		DatabaseQueryTimeout:     queryTimeout,
		OtelServiceName:          otelService,
		OtelExporterOtlpEndpoint: os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		OtelExporterOtlpProtocol: protocol,
		OtelResourceAttributes:   os.Getenv("OTEL_RESOURCE_ATTRIBUTES"),
		OtelTracesSampler:        sampler,
	}, nil
}

func parseDurationEnv(key, defaultVal string) (time.Duration, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		valStr = defaultVal
	}
	d, err := time.ParseDuration(valStr)
	if err != nil {
		return 0, fmt.Errorf("некорректное значение переменной %s: %w", key, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("переменная %s должна быть положительной продолжительностью, получено: %s", key, valStr)
	}
	return d, nil
}

func parsePositiveIntEnv(key string, defaultVal int32) (int32, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal, nil
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil || val <= 0 {
		return 0, fmt.Errorf("некорректное значение переменной %s: должно быть положительным целым числом, получено: %q", key, valStr)
	}
	return int32(val), nil
}

func parseNonNegativeIntEnv(key string, defaultVal int32) (int32, error) {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultVal, nil
	}
	val, err := strconv.ParseInt(valStr, 10, 32)
	if err != nil || val < 0 {
		return 0, fmt.Errorf("некорректное значение переменной %s: должно быть неотрицательным целым числом, получено: %q", key, valStr)
	}
	return int32(val), nil
}

func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("некорректное значение переменной LOG_LEVEL: %q (поддерживаются debug, info, warn, error)", level)
	}
}
