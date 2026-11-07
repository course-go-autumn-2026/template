package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/XrystikLK/template/internal/config"
	"github.com/XrystikLK/template/internal/db"
	api "github.com/XrystikLK/template/internal/generated"
	"github.com/XrystikLK/template/internal/handler"
	"github.com/XrystikLK/template/internal/repository"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.ParsedLogLevel,
	}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.InitPool(ctx, cfg)
	if err != nil {
		slog.Error("failed to initialize db pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	txManager := db.NewTxManager(pool)
	tripRepo := repository.NewTripRepository(pool, cfg.DatabaseQueryTimeout)
	tripHandler := handler.NewTripHandler(tripRepo, txManager, pool)

	r := chi.NewRouter()
	api.HandlerWithOptions(tripHandler, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handler.WriteProblem(w, http.StatusBadRequest, "invalid_request", "Invalid request", err.Error(), r.URL.Path)
		},
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	slog.Info("server starting", "addr", cfg.HTTPAddr)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
}
