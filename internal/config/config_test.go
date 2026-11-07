package config_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/XrystikLK/template/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Load_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestConfig_Load_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("DATABASE_MAX_CONNS", "")
	t.Setenv("DATABASE_MIN_CONNS", "")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, slog.LevelInfo, cfg.ParsedLogLevel)
	assert.Equal(t, int32(10), cfg.DatabaseMaxConns)
	assert.Equal(t, int32(2), cfg.DatabaseMinConns)
	assert.Equal(t, 5*time.Second, cfg.DatabaseConnectTimeout)
	assert.Equal(t, 3*time.Second, cfg.DatabaseQueryTimeout)
	assert.Equal(t, 30*time.Minute, cfg.DatabaseMaxConnLifetime)
	assert.Equal(t, 10*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, 5*time.Second, cfg.HTTPReadTimeout)
	assert.Equal(t, 2*time.Second, cfg.HTTPReadHeaderTimeout)
	assert.Equal(t, 5*time.Second, cfg.HTTPWriteTimeout)
	assert.Equal(t, 30*time.Second, cfg.HTTPIdleTimeout)
}

func TestConfig_Load_CustomValid(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("DATABASE_MAX_CONNS", "25")
	t.Setenv("DATABASE_MIN_CONNS", "5")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "10s")
	t.Setenv("DATABASE_QUERY_TIMEOUT", "4s")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "1h")
	t.Setenv("SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("HTTP_READ_TIMEOUT", "8s")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.HTTPAddr)
	assert.Equal(t, slog.LevelDebug, cfg.ParsedLogLevel)
	assert.Equal(t, int32(25), cfg.DatabaseMaxConns)
	assert.Equal(t, int32(5), cfg.DatabaseMinConns)
	assert.Equal(t, 10*time.Second, cfg.DatabaseConnectTimeout)
	assert.Equal(t, 4*time.Second, cfg.DatabaseQueryTimeout)
	assert.Equal(t, time.Hour, cfg.DatabaseMaxConnLifetime)
	assert.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
	assert.Equal(t, 8*time.Second, cfg.HTTPReadTimeout)
}

func TestConfig_Load_InvalidMaxConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("DATABASE_MAX_CONNS", "zero")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_MAX_CONNS")

	t.Setenv("DATABASE_MAX_CONNS", "-5")
	_, err = config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_MAX_CONNS")
}

func TestConfig_Load_MinExceedsMaxConns(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("DATABASE_MAX_CONNS", "5")
	t.Setenv("DATABASE_MIN_CONNS", "10")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_MIN_CONNS")
}

func TestConfig_Load_InvalidDuration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "not-a-duration")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_CONNECT_TIMEOUT")

	t.Setenv("DATABASE_CONNECT_TIMEOUT", "-1s")
	_, err = config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_CONNECT_TIMEOUT")
}

func TestConfig_Load_InvalidLogLevel(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("LOG_LEVEL", "superverbose")
	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOG_LEVEL")
}
