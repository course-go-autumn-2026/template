package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/XrystikLK/template/internal/config"
	"github.com/XrystikLK/template/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	app, err := container.New(ctx, cfg)
	if err != nil {
		stop()
		log.Fatal(err)
	}
	defer stop()

	app.Logger().Info("service starting", slog.String("addr", app.Addr()))

	go func() {
		if err := app.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			app.Logger().Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	app.Logger().Info("shutting down gracefully")

	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		app.Logger().Error("server shutdown error", slog.Any("error", err))
	}
}
