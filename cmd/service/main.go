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

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c, err := container.New(rootCtx, cfg)
	if err != nil {
		slog.Error("container init error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	c.Poller.Start(rootCtx)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           c.Router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		c.Logger.Info("starting http server", slog.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-rootCtx.Done():
		c.Logger.Info("shutdown signal received")
	case srvErr := <-errCh:
		if srvErr != nil {
			c.Logger.Error("http server error", slog.String("error", srvErr.Error()))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// 1. Останавливаем HTTP-сервер и дожидаемся текущих запросов
	if err := srv.Shutdown(shutdownCtx); err != nil {
		c.Logger.Error("http server shutdown error", slog.String("error", err.Error()))
	}

	// 2–3. Останавливаем сканер и даём воркерам доработать очередь в пределах SHUTDOWN_TIMEOUT
	if err := c.Poller.Shutdown(shutdownCtx); err != nil {
		c.Logger.Error("position poller shutdown error", slog.String("error", err.Error()))
	}

	// 4. Закрываем HTTP-клиент и пул БД
	c.PushClient.Close()
	c.Pool.Close()

	if c.TracerProvider != nil {
		_ = c.TracerProvider.Shutdown(shutdownCtx)
	}
	if c.MeterProvider != nil {
		_ = c.MeterProvider.Shutdown(shutdownCtx)
	}

	c.Logger.Info("service stopped cleanly")
}
