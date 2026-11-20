package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/n1ckerr0r/avito-labs/internal/config"
	"github.com/n1ckerr0r/avito-labs/internal/container"
)

func main() {
	if err := run(); err != nil {
		slog.Error("trip service startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := container.New(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("build container: %w", err)
	}

	errCh := make(chan error, 1)
	go func() { errCh <- app.Server.ListenAndServe() }()

	app.Logger.Info("trip service started", "address", cfg.HTTPAddr)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve HTTP: %w", err)
		}

		return nil
	case <-signalCtx.Done():
		app.Logger.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := app.Server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		if err := app.Close(shutdownCtx); err != nil {
			return fmt.Errorf("close dependencies: %w", err)
		}

		return nil
	}
}
