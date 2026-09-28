package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr              string
	HTTPReadTimeout       time.Duration
	HTTPReadHeaderTimeout time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration

	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (Config, error) {
	var cfg Config
	var err error

	cfg.HTTPAddr, err = readString("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTPReadTimeout, err = readDuration("HTTP_READ_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTPReadHeaderTimeout, err = readDuration("HTTP_READ_HEADER_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTPWriteTimeout, err = readDuration("HTTP_WRITE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.HTTPIdleTimeout, err = readDuration("HTTP_IDLE_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.LogLevel, err = readString("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	cfg.ShutdownTimeout, err = readDuration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseURL, err = readString("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConns, err = readInt32("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMinConns, err = readInt32("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseMaxConnLifetime, err = readDuration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseConnectTimeout, err = readDuration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	cfg.DatabaseQueryTimeout, err = readDuration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}

func readString(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%s must be set and not empty", name)
	}
	return value, nil
}

func readDuration(name string) (time.Duration, error) {
	raw, err := readString(name)
	if err != nil {
		return 0, err
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return value, nil
}

func readInt32(name string) (int32, error) {
	raw, err := readString(name)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return int32(value), nil
}

func (c Config) Validate() error {
	_, port, err := net.SplitHostPort(c.HTTPAddr)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR must have host:port format: %w", err)
	}

	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("HTTP_ADDR port must be a number: %w", err)
	}
	if portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("HTTP_ADDR port must be between 1 and 65535")
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}

	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL must not be empty")
	}
	if c.DatabaseMaxConns <= 0 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if c.DatabaseMinConns < 0 || c.DatabaseMinConns > c.DatabaseMaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS must be between 0 and DATABASE_MAX_CONNS")
	}

	durations := []struct {
		name  string
		value time.Duration
	}{
		{"HTTP_READ_TIMEOUT", c.HTTPReadTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", c.HTTPReadHeaderTimeout},
		{"HTTP_WRITE_TIMEOUT", c.HTTPWriteTimeout},
		{"HTTP_IDLE_TIMEOUT", c.HTTPIdleTimeout},
		{"SHUTDOWN_TIMEOUT", c.ShutdownTimeout},
		{"DATABASE_MAX_CONN_LIFETIME", c.DatabaseMaxConnLifetime},
		{"DATABASE_CONNECT_TIMEOUT", c.DatabaseConnectTimeout},
		{"DATABASE_QUERY_TIMEOUT", c.DatabaseQueryTimeout},
	}

	for _, duration := range durations {
		if duration.value <= 0 {
			return fmt.Errorf("%s must be positive", duration.name)
		}
	}

	return nil
}
