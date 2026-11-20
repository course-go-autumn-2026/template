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

	"github.com/George1487/MalyginGeorge/internal/config"
	"github.com/George1487/MalyginGeorge/internal/container"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	appCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	c, err := container.New(appCtx, cfg)
	if err != nil {
		return fmt.Errorf("init container: %w", err)
	}
	defer func() {
		if c.Shutdown != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()
			_ = c.Shutdown(shutdownCtx)
		}
	}()

	scannerCtx, scannerCancel := context.WithCancel(context.Background())
	defer scannerCancel()

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	c.WorkerPool.Start(cfg.PositionWorkers, workerCtx, cfg.PushRequestTimeout)

	scannerErr := make(chan error, 1)
	go func() {
		c.Logger.Info("starting positions scanner",
			"interval", cfg.PositionCheckInterval,
			"stale_after", cfg.PositionStaleAfter,
			"batch_size", cfg.PositionScanBatchSize,
		)
		scannerErr <- c.Scaner.Run(scannerCtx, cfg.PositionCheckInterval, cfg.PositionStaleAfter, cfg.PositionScanBatchSize)
	}()

	serverErr := make(chan error, 1)
	go func() {
		c.Logger.Info("starting HTTP server", "addr", cfg.HTTPAddr)
		serverErr <- c.HTTPServer.ListenAndServe()
	}()

	select {
	case err = <-serverErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case err = <-scannerErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("positions scanner: %w", err)
		}
		return nil
	case <-appCtx.Done():
	}

	// Graceful Shutdown (порядок по п. 5.7 ТЗ):
	// 1. HTTP-сервер перестаёт принимать запросы и дожидается текущих;
	// 2. сканер останавливается — новые задачи не появляются;
	// 3. воркеры дорабатывают то, что уже в очереди, в пределах общего SHUTDOWN_TIMEOUT;
	// 4. закрываются HTTP-клиент и пул БД.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	// Шаг 1: Остановка HTTP-сервера
	c.Logger.Info("1. shutting down HTTP server")
	if err = c.HTTPServer.Shutdown(shutdownCtx); err != nil {
		closeErr := c.HTTPServer.Close()
		if closeErr != nil {
			return errors.Join(
				fmt.Errorf("shutdown HTTP server: %w", err),
				fmt.Errorf("force close HTTP server: %w", closeErr),
			)
		}
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	// Шаг 2: Остановка сканера
	c.Logger.Info("2. stopping positions scanner")
	scannerCancel()
	<-scannerErr

	// Шаг 3: Воркеры дорабатывают то, что уже в очереди, в пределах SHUTDOWN_TIMEOUT
	c.Logger.Info("3. waiting for worker pool to drain queue")
	workersDone := make(chan struct{})
	go func() {
		c.WorkerPool.Wait()
		close(workersDone)
	}()

	select {
	case <-workersDone:
		c.Logger.Info("all workers finished remaining tasks")
	case <-shutdownCtx.Done():
		c.Logger.Warn("shutdown timeout reached, cancelling worker tasks")
		workerCancel()
		<-workersDone
	}

	// Шаг 4: Закрытие БД и телеметрии
	c.Logger.Info("4. closing storage and telemetry")
	if c.Shutdown != nil {
		if err := c.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown resources: %w", err)
		}
	}

	return nil
}
