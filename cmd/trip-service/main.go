package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/freyliee/template/internal/config"
	"github.com/freyliee/template/internal/container"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx := context.Background()

	app, err := container.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}

	app.Start(ctx)

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	failures := make(chan error, 1)

	go func() {
		app.Logger.InfoContext(ctx, "service started", "address", cfg.HTTPAddr)
		failures <- app.Server.ListenAndServe()
	}()

	var result error

	select {
	case <-signalCtx.Done():
		stop()
	case err := <-failures:
		if !errors.Is(err, http.ErrServerClosed) {
			result = err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, cfg.ShutdownTimeout)
	defer cancel()

	watchdog := time.AfterFunc(cfg.ShutdownTimeout, func() { app.Logger.ErrorContext(ctx, "shutdown timeout exceeded"); os.Exit(1) })
	defer watchdog.Stop()

	app.Logger.InfoContext(ctx, "service stopping")

	if err := app.Server.Shutdown(shutdownCtx); err != nil {
		result = errors.Join(result, err, app.Server.Close())
	}

	result = errors.Join(result, app.Close(shutdownCtx))
	app.Logger.InfoContext(ctx, "service stopped")

	return result
}
