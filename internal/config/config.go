package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	DatabaseURL             string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration

	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	HTTPIdleTimeout  time.Duration
	ReadyTimeout     time.Duration
}

func Load() (Config, error) {
	var cfg Config
	var err error

	if cfg.HTTPAddr, err = requiredEnv("HTTP_ADDR"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseURL, err = requiredEnv("DATABASE_URL"); err != nil {
		return Config{}, err
	}
	if cfg.LogLevel, err = requiredEnv("LOG_LEVEL"); err != nil {
		return Config{}, err
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}

	if cfg.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseConnectTimeout, err = durationEnv("DATABASE_CONNECT_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseQueryTimeout, err = durationEnv("DATABASE_QUERY_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.DatabaseMaxConnLifetime, err = durationEnv("DATABASE_MAX_CONN_LIFETIME"); err != nil {
		return Config{}, err
	}

	if cfg.HTTPReadTimeout, err = durationEnv("HTTP_READ_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.HTTPWriteTimeout, err = durationEnv("HTTP_WRITE_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.HTTPIdleTimeout, err = durationEnv("HTTP_IDLE_TIMEOUT"); err != nil {
		return Config{}, err
	}
	if cfg.ReadyTimeout, err = durationEnv("READY_TIMEOUT"); err != nil {
		return Config{}, err
	}

	maxConns, err := intEnv("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}
	minConns, err := intEnv("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	if maxConns < 1 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be greater than zero")
	}
	if minConns < 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS cannot be negative")
	}
	if minConns > maxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS cannot exceed DATABASE_MAX_CONNS")
	}

	cfg.DatabaseMaxConns = int32(maxConns)
	cfg.DatabaseMinConns = int32(minConns)

	return cfg, nil
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

func durationEnv(key string) (time.Duration, error) {
	value, err := requiredEnv(key)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", key)
	}

	return duration, nil
}

func intEnv(key string) (int, error) {
	value, err := requiredEnv(key)
	if err != nil {
		return 0, err
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return number, nil
}