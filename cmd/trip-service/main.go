package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/whxtelyy/template/internal/config"
	"github.com/whxtelyy/template/internal/container"
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
		return fmt.Errorf("load configuration: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	app, err := container.New(ctx, cfg)
	if err != nil {
		return err
	}
	var closeCtx context.Context
	var closeCancel context.CancelFunc
	defer func() {
		if closeCtx != nil {
			app.Close(closeCtx)
			closeCancel()
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		app.Close(ctx)
	}()
	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	serveResult := make(chan error, 1)
	go func() { serveResult <- app.Server.Serve(listener) }()
	app.Logger.Info("http server started", "addr", cfg.HTTPAddr)
	select {
	case err := <-serveResult:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		app.Logger.Info("http server shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		closeCtx = shutdownCtx
		closeCancel = cancel
		if err := app.Server.Shutdown(shutdownCtx); err != nil {
			_ = app.Server.Close()
			return fmt.Errorf("shutdown HTTP: %w", err)
		}
		if err := <-serveResult; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		app.Logger.Info("http server stopped")
		return nil
	}
}
