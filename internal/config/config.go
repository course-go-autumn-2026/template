package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr               string
	DatabaseURL            string
	LogLevel               string
	ShutdownTimeout        time.Duration
	DatabaseMaxConns       int32
	DatabaseMinConns       int32
	DatabaseConnectTimeout time.Duration
	DatabaseQueryTimeout   time.Duration
	DatabaseMaxConnLife    time.Duration
	OtelEndpoint           string
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	shutdownTimeout, err := parseDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}

	maxConns, err := parseInt32Env("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}

	minConns, err := parseInt32Env("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}

	connTimeout, err := parseDurationEnv("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	queryTimeout, err := parseDurationEnv("DATABASE_QUERY_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}

	maxConnLife, err := parseDurationEnv("DATABASE_MAX_CONN_LIFETIME", time.Hour)
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPAddr:               getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:            dbURL,
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:        shutdownTimeout,
		DatabaseMaxConns:       maxConns,
		DatabaseMinConns:       minConns,
		DatabaseConnectTimeout: connTimeout,
		DatabaseQueryTimeout:   queryTimeout,
		DatabaseMaxConnLife:    maxConnLife,
		OtelEndpoint:           getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4318"),
	}, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parseInt32Env(k string, def int32) (int32, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(n), nil
}

func parseDurationEnv(k string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(k)
	if v == "" {
		return def, nil
	}
	return time.ParseDuration(v)
}
