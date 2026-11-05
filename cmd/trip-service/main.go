package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/eternal-despair/template/internal/config"
	"github.com/eternal-despair/template/internal/container"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	app, err := container.New(ctx, cfg)
	if err != nil {
		cancel()
		log.Fatalf("failed to initialize container: %v", err)
	}
	defer cancel()
	defer app.Close()

	serverErrCh := make(chan error, 1)
	go func() {
		if err := app.Run(); err != nil {
			serverErrCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		log.Printf("server error: %v", err)
	case sig := <-sigCh:
		log.Printf("received signal %v, initiating graceful shutdown (budget: %v)...", sig, cfg.ShutdownTimeout)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer shutdownCancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		log.Printf("server graceful shutdown failed: %v", err)
	}
}
