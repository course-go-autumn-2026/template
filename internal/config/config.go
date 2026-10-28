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
	Database        DatabaseConfig
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

func Load() (Config, error) {
	httpAddr, err := required("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	logLevel, err := required("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := duration("SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	databaseURL, err := required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	maxConns, err := int32Value("DATABASE_MAX_CONNS")
	if err != nil {
		return Config{}, err
	}

	minConns, err := int32Value("DATABASE_MIN_CONNS")
	if err != nil {
		return Config{}, err
	}

	if minConns > maxConns {
		return Config{}, fmt.Errorf(
			"DATABASE_MIN_CONNS cannot be greater than DATABASE_MAX_CONNS",
		)
	}

	maxConnLifetime, err := duration("DATABASE_MAX_CONN_LIFETIME")
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := duration("DATABASE_CONNECT_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	queryTimeout, err := duration("DATABASE_QUERY_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:        httpAddr,
		LogLevel:        logLevel,
		ShutdownTimeout: shutdownTimeout,
		Database: DatabaseConfig{
			URL:             databaseURL,
			MaxConns:        maxConns,
			MinConns:        minConns,
			MaxConnLifetime: maxConnLifetime,
			ConnectTimeout:  connectTimeout,
			QueryTimeout:    queryTimeout,
		},
	}, nil
}

func required(name string) (string, error) {
	value, exists := os.LookupEnv(name)
	if !exists || value == "" {
		return "", fmt.Errorf("environment variable %s is required", name)
	}

	return value, nil
}

func duration(name string) (time.Duration, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid duration %q: %w", name, value, err)
	}

	if result <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", name)
	}

	return result, nil
}

func int32Value(name string) (int32, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}

	result, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s has invalid integer %q: %w", name, value, err)
	}

	if result < 0 {
		return 0, fmt.Errorf("%s cannot be negative", name)
	}

	return int32(result), nil
}
