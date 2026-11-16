package container_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/XrystikLK/template/internal/config"
	"github.com/XrystikLK/template/internal/container"
)

func TestContainer_Shutdown_RespectsContextDeadline(t *testing.T) {
	cfg := &config.Config{
		HTTPAddr:        ":0",
		HTTPReadTimeout: time.Second,
		ShutdownTimeout: 50 * time.Millisecond,
		LogLevel:        "info",
	}

	app, err := container.New(context.Background(), cfg)
	if err != nil {
		t.Skip("skipping integration container test without real database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_ = app.Shutdown(ctx)
	duration := time.Since(start)

	assert.Less(t, duration, 200*time.Millisecond)
}
