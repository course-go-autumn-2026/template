package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
	DatabaseMaxConnLifetime time.Duration
}

func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("required environment variable DATABASE_URL is missing")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	shutdownTimeout, err := time.ParseDuration(getEnvWithDefault("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		shutdownTimeout = 10 * time.Second
	}

	// Читаем параметры пула и таймауты
	maxConns := int32(getEnvInt("DATABASE_MAX_CONNS", 10))
	minConns := int32(getEnvInt("DATABASE_MIN_CONNS", 2))

	connectTimeout, _ := time.ParseDuration(getEnvWithDefault("DATABASE_CONNECT_TIMEOUT", "5s"))
	queryTimeout, _ := time.ParseDuration(getEnvWithDefault("DATABASE_QUERY_TIMEOUT", "3s"))
	maxConnLifetime, _ := time.ParseDuration(getEnvWithDefault("DATABASE_MAX_CONN_LIFETIME", "30m"))

	return &Config{
		HTTPAddr:                httpAddr,
		DatabaseURL:             dbURL,
		LogLevel:                logLevel,
		ShutdownTimeout:         shutdownTimeout,
		DatabaseMaxConns:        maxConns,
		DatabaseMinConns:        minConns,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
		DatabaseMaxConnLifetime: maxConnLifetime,
	}, nil
}

func getEnvWithDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}
