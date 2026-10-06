package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{}

	// Обязательные переменные (без них сервис не стартует)
	var err error
	cfg.HTTPAddr, err = getEnvRequired("HTTP_ADDR")
	if err != nil {
		return nil, err
	}

	cfg.DatabaseURL, err = getEnvRequired("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	// Переменные с безопасными дефолтами
	cfg.LogLevel = getEnvOrDefault("LOG_LEVEL", "info")

	shutdownSec, err := getDurationOrDefault("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.ShutdownTimeout = shutdownSec

	// Параметры пула БД
	maxConns, err := getInt32OrDefault("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMaxConns = maxConns

	minConns, err := getInt32OrDefault("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMinConns = minConns

	lifetime, err := getDurationOrDefault("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseMaxConnLifetime = lifetime

	connTimeout, err := getDurationOrDefault("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseConnectTimeout = connTimeout

	queryTimeout, err := getDurationOrDefault("DATABASE_QUERY_TIMEOUT", 3*time.Second)
	if err != nil {
		return nil, err
	}
	cfg.DatabaseQueryTimeout = queryTimeout

	return cfg, nil
}

func getEnvRequired(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return val, nil
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getDurationOrDefault(key string, fallback time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %q: %w", key, err)
	}
	return d, nil
}

func getInt32OrDefault(key string, fallback int32) (int32, error) {
	val := os.Getenv(key)
	if val == "" {
		return fallback, nil
	}
	n, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %q: %w", key, err)
	}
	return int32(n), nil
}