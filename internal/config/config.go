package config

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTP            HTTP
	Database        Database
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	IdempotencyTTL  time.Duration
}

type HTTP struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type Database struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

// Load проверяет конфигурацию до открытия соединений и запуска сервера.
func Load() (Config, error) {
	var cfg Config
	var err error
	cfg.HTTP.Addr, err = required("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}
	_, port, err := net.SplitHostPort(cfg.HTTP.Addr)
	if err != nil {
		return Config{}, fmt.Errorf("HTTP_ADDR must have host:port format")
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return Config{}, fmt.Errorf("HTTP_ADDR must contain a port between 1 and 65535")
	}

	cfg.Database.URL, err = required("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}
	databaseURL, err := url.Parse(cfg.Database.URL)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") || databaseURL.Hostname() == "" || strings.Trim(databaseURL.Path, "/") == "" {
		// Не включаем значение переменной в ошибку: URL может содержать пароль.
		return Config{}, fmt.Errorf("DATABASE_URL must be a PostgreSQL URL with host and database name")
	}

	level, err := required("LOG_LEVEL")
	if err != nil {
		return Config{}, err
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(level)); err != nil {
		return Config{}, fmt.Errorf("LOG_LEVEL must be a valid slog level")
	}

	for _, setting := range []struct {
		name  string
		value *time.Duration
	}{
		{"SHUTDOWN_TIMEOUT", &cfg.ShutdownTimeout},
		{"HTTP_READ_TIMEOUT", &cfg.HTTP.ReadTimeout},
		{"HTTP_READ_HEADER_TIMEOUT", &cfg.HTTP.ReadHeaderTimeout},
		{"HTTP_WRITE_TIMEOUT", &cfg.HTTP.WriteTimeout},
		{"HTTP_IDLE_TIMEOUT", &cfg.HTTP.IdleTimeout},
		{"DATABASE_MAX_CONN_LIFETIME", &cfg.Database.MaxConnLifetime},
		{"DATABASE_CONNECT_TIMEOUT", &cfg.Database.ConnectTimeout},
		{"DATABASE_QUERY_TIMEOUT", &cfg.Database.QueryTimeout},
		{"IDEMPOTENCY_TTL", &cfg.IdempotencyTTL},
	} {
		value, err := required(setting.name)
		if err != nil {
			return Config{}, err
		}
		duration, err := time.ParseDuration(value)
		if err != nil || duration <= 0 {
			return Config{}, fmt.Errorf("%s must be a positive duration (for example, 5s)", setting.name)
		}
		*setting.value = duration
	}

	cfg.Database.MaxConns, err = connections("DATABASE_MAX_CONNS", 1)
	if err != nil {
		return Config{}, err
	}
	cfg.Database.MinConns, err = connections("DATABASE_MIN_CONNS", 0)
	if err != nil {
		return Config{}, err
	}
	if cfg.Database.MinConns > cfg.Database.MaxConns {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	return cfg, nil
}

func required(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func connections(name string, minimum int32) (int32, error) {
	value, err := required(name)
	if err != nil {
		return 0, err
	}
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil || number < int64(minimum) {
		return 0, fmt.Errorf("%s must be an integer between %d and 2147483647", name, minimum)
	}
	return int32(number), nil
}
