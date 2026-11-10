package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/murmyauuu/template/internal/config"
	"github.com/murmyauuu/template/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c, err := container.New(ctx, cfg)
	if err != nil {
		slog.Error("failed to build container", "error", err)
		os.Exit(1)
	}
	logger := c.Logger

	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- c.Server.ListenAndServe()
	}()
	logger.Info("HTTP server starting", "addr", cfg.HTTP.Addr)

	var serveErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			serveErr = fmt.Errorf("serve HTTP: %w", err)
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()
	if err := c.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
	if serveErr != nil {
		logger.Error("service stopped with an error", "error", serveErr)
		os.Exit(1)
	}
	logger.Info("service stopped")
}
