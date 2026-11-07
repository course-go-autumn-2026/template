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
}

func Load() (*Config, error) {
	httpAddr := getEnv("HTTP_ADDR", ":8080")
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
		HTTPAddr:               httpAddr,
		DatabaseURL:            dbURL,
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:        shutdownTimeout,
		DatabaseMaxConns:       maxConns,
		DatabaseMinConns:       minConns,
		DatabaseConnectTimeout: connTimeout,
		DatabaseQueryTimeout:   queryTimeout,
		DatabaseMaxConnLife:    maxConnLife,
	}, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseInt32Env(key string, defaultVal int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	parsed, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return int32(parsed), nil
}

func parseDurationEnv(key string, defaultVal time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return d, nil
}
