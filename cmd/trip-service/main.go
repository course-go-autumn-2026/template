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
	if err := run(); err != nil {
		slog.Error("service stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	c, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to build container: %w", err)
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
		return serveErr
	}
	logger.Info("service stopped")
	return nil
}
