package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                string
	LogLevel                string
	ShutdownTimeout         time.Duration
	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:                getEnv("HTTP_ADDR", ":8080"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:         getEnvDuration("SHUTDOWN_TIMEOUT", 5*time.Second),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		DatabaseMaxConns:        getEnvInt32("DATABASE_MAX_CONNS", 10),
		DatabaseMinConns:        getEnvInt32("DATABASE_MIN_CONNS", 2),
		DatabaseConnectTimeout:  getEnvDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second),
		DatabaseQueryTimeout:    getEnvDuration("DATABASE_QUERY_TIMEOUT", 3*time.Second),
		DatabaseMaxConnLifetime: getEnvDuration("DATABASE_MAX_CONN_LIFETIME", 1*time.Hour),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getEnvInt32(key string, fallback int32) int32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(n)
}
