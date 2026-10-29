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
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:                getEnv("HTTP_ADDR", ":8080"),
		LogLevel:                getEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:         getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		DatabaseURL:             os.Getenv("DATABASE_URL"),
		DatabaseMaxConns:        int32(getInt("DATABASE_MAX_CONNS", 10)),
		DatabaseMinConns:        int32(getInt("DATABASE_MIN_CONNS", 2)),
		DatabaseMaxConnLifetime: getDuration("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute),
		DatabaseConnectTimeout:  getDuration("DATABASE_CONNECT_TIMEOUT", 5*time.Second),
		DatabaseQueryTimeout:    getDuration("DATABASE_QUERY_TIMEOUT", 3*time.Second),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}