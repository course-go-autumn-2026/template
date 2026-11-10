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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/murmyauuu/template/internal/config"
	"github.com/murmyauuu/template/internal/database"
	"github.com/murmyauuu/template/internal/httpapi"
	"github.com/murmyauuu/template/internal/trip"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "trip-service")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})).With("service", "trip-service")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg, logger); err != nil {
		logger.Error("service stopped with an error", "error", err)
		os.Exit(1)
	}
	logger.Info("service stopped")
}

func run(ctx context.Context, cfg config.Config, logger *slog.Logger) error {
	pool, err := database.Open(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	trips := trip.NewService(
		trip.NewRepository(pool, cfg.Database.QueryTimeout),
		trip.NewHistoryRepository(pool, cfg.Database.QueryTimeout),
		database.NewTxManager(pool, cfg.Database.QueryTimeout),
		trip.NewIdempotencyRepository(pool, cfg.Database.QueryTimeout),
		cfg.IdempotencyTTL,
	)
	server := httpapi.New(cfg.HTTP, pool, trips, cfg.Database.QueryTimeout, logger)
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
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
	return errors.Join(serveErr, shutdown(shutdownCtx, server, pool))
}

func shutdown(ctx context.Context, server *http.Server, pool *pgxpool.Pool) error {
	finished := make(chan error, 1)
	go func() {
		err := server.Shutdown(ctx)
		if err != nil {
			// После истечения бюджета прекращаем обработку оставшихся запросов.
			_ = server.Close()
		}
		// Close ждёт возврата соединений; ожидание входит в общий бюджет.
		pool.Close()
		finished <- err
	}()
	select {
	case err := <-finished:
		if err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown deadline exceeded: %w", ctx.Err())
	}
}
