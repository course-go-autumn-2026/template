package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Configuration struct {
	PostgreSQL PostgreSQL
	Server     Server
	LogLevel   string
}

type Server struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxHeaderBytes    int
}

type PostgreSQL struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

func Load() (*Configuration, error) {
	loader := envLoader{}

	cfg := &Configuration{
		Server: Server{
			Addr:              loader.string("HTTP_ADDR"),
			ReadTimeout:       loader.duration("HTTP_READ_TIMEOUT"),
			ReadHeaderTimeout: loader.duration("HTTP_READ_HEADER_TIMEOUT"),
			WriteTimeout:      loader.duration("HTTP_WRITE_TIMEOUT"),
			IdleTimeout:       loader.duration("HTTP_IDLE_TIMEOUT"),
			ShutdownTimeout:   loader.duration("SHUTDOWN_TIMEOUT"),
			MaxHeaderBytes:    loader.integer("HTTP_MAX_HEADER_BYTES"),
		},
		PostgreSQL: PostgreSQL{
			URL:             loader.string("DATABASE_URL"),
			MaxConns:        loader.int32("DATABASE_MAX_CONNS"),
			MinConns:        loader.int32("DATABASE_MIN_CONNS"),
			MaxConnLifetime: loader.duration("DATABASE_MAX_CONN_LIFETIME"),
			ConnectTimeout:  loader.duration("DATABASE_CONNECT_TIMEOUT"),
			QueryTimeout:    loader.duration("DATABASE_QUERY_TIMEOUT"),
		},
		LogLevel: strings.ToLower(loader.string("LOG_LEVEL")),
	}

	if loader.err != nil {
		return nil, loader.err
	}

	if cfg.Server.MaxHeaderBytes <= 0 {
		return nil, fmt.Errorf("HTTP_MAX_HEADER_BYTES must be positive")
	}
	if cfg.PostgreSQL.MaxConns <= 0 {
		return nil, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}
	if cfg.PostgreSQL.MinConns < 0 {
		return nil, fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if cfg.PostgreSQL.MinConns > cfg.PostgreSQL.MaxConns {
		return nil, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}

	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return nil, fmt.Errorf(
			"LOG_LEVEL must be one of debug, info, warn or error",
		)
	}

	return cfg, nil
}

type envLoader struct {
	err error
}

func (l *envLoader) string(name string) string {
	if l.err != nil {
		return ""
	}

	value, ok := os.LookupEnv(name)
	value = strings.TrimSpace(value)
	if !ok || value == "" {
		l.err = fmt.Errorf("required environment variable %s is not set", name)
		return ""
	}

	return value
}

func (l *envLoader) duration(name string) time.Duration {
	value := l.string(name)
	if l.err != nil {
		return 0
	}

	result, err := time.ParseDuration(value)
	if err != nil {
		l.err = fmt.Errorf("parse %s: %w", name, err)
		return 0
	}
	if result <= 0 {
		l.err = fmt.Errorf("%s must be positive", name)
		return 0
	}

	return result
}

func (l *envLoader) int32(name string) int32 {
	value := l.string(name)
	if l.err != nil {
		return 0
	}

	result, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		l.err = fmt.Errorf("parse %s: %w", name, err)
		return 0
	}

	return int32(result)
}

func (l *envLoader) integer(name string) int {
	value := l.string(name)
	if l.err != nil {
		return 0
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		l.err = fmt.Errorf("parse %s: %w", name, err)
		return 0
	}

	return result
}
