package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xotapy-main/courseGo/internal/config"
	"github.com/xotapy-main/courseGo/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := container.New(ctx, cfg)
	if err != nil {
		slog.Error("container init error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer c.Pool.Close()

	if c.TracerProvider != nil {
		defer func() { _ = c.TracerProvider.Shutdown(context.Background()) }()
	}
	if c.MeterProvider != nil {
		defer func() { _ = c.MeterProvider.Shutdown(context.Background()) }()
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           c.Router,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() {
		c.Logger.Info("server starting", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		c.Logger.Error("server error", slog.String("error", err.Error()))
		os.Exit(1)
	case sig := <-shutdown:
		c.Logger.Info("shutting down", slog.String("signal", sig.String()))

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			c.Logger.Error("forced shutdown", slog.String("error", err.Error()))
			os.Exit(1)
		}
		c.Logger.Info("server stopped")
	}
}
