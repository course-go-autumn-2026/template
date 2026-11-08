package container

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/exaring/otelpgx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/metric/noop"

	"github.com/whxtelyy/template/internal/config"
	api "github.com/whxtelyy/template/internal/generated"
	httpapi "github.com/whxtelyy/template/internal/http"
	"github.com/whxtelyy/template/internal/http/middleware"
	"github.com/whxtelyy/template/internal/infra/postgres"
	"github.com/whxtelyy/template/internal/infra/postgres/transaction"
	"github.com/whxtelyy/template/internal/infra/telemetry"
	repository "github.com/whxtelyy/template/internal/repository/trip"
	"github.com/whxtelyy/template/internal/usecase/trip/create"
	"github.com/whxtelyy/template/internal/usecase/trip/finish"
	"github.com/whxtelyy/template/internal/usecase/trip/get"
)

type App struct {
	Server    *http.Server
	Logger    *slog.Logger
	pool      *pgxpool.Pool
	telemetry *telemetry.Providers
}

func (a *App) Close(ctx context.Context) {
	a.pool.Close()
	if err := a.telemetry.Shutdown(ctx); err != nil {
		a.Logger.Error("telemetry shutdown failed", "error", err)
	}
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logger := slog.New(telemetry.NewTraceHandler(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))).With("service", "trip-service")
	providers, err := telemetry.New(ctx, logger)
	if err != nil {
		return nil, err
	}
	poolCfg, err := pgxpool.ParseConfig(cfg.Database.URL)
	if err != nil {
		_ = providers.Shutdown(context.Background())
		return nil, fmt.Errorf("parse database configuration: %w", err)
	}
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.MinConns = cfg.Database.MinConns
	poolCfg.MaxConnLifetime = cfg.Database.MaxConnLifetime
	poolCfg.ConnConfig.ConnectTimeout = cfg.Database.ConnectTimeout
	queryTracer, err := postgres.NewQueryTracer(otelpgx.NewTracer(otelpgx.WithTracerProvider(providers.Trace), otelpgx.WithMeterProvider(noop.NewMeterProvider())), providers.Metric.Meter("trip-service/db"))
	if err != nil {
		_ = providers.Shutdown(context.Background())
		return nil, err
	}
	poolCfg.ConnConfig.Tracer = queryTracer
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		_ = providers.Shutdown(context.Background())
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, cfg.Database.ConnectTimeout)
	err = pool.Ping(pingCtx)
	cancel()
	if err != nil {
		pool.Close()
		_ = providers.Shutdown(context.Background())
		return nil, fmt.Errorf("ping database: %w", err)
	}
	repo := repository.NewTripRepository(pool, cfg.Database.QueryTimeout)
	txManager := transaction.NewManager(pool, pgx.ReadCommitted, cfg.Database.QueryTimeout, logger)
	handler := httpapi.NewHandler(create.New(repo, txManager, providers), get.New(repo), finish.New(repo, txManager, providers), pool, logger, cfg.Database.QueryTimeout)
	metrics, err := middleware.Metrics(providers.Metric.Meter("trip-service/http"))
	if err != nil {
		pool.Close()
		_ = providers.Shutdown(context.Background())
		return nil, err
	}
	router := chi.NewRouter()
	router.Use(middleware.Recover(logger), middleware.Tracing(providers.Trace), middleware.Logging(logger), metrics)
	httpHandler := api.HandlerWithOptions(handler, api.ChiServerOptions{BaseRouter: router, ErrorHandlerFunc: handler.GeneratedErrorHandler})
	server := &http.Server{
		Addr: cfg.HTTPAddr, Handler: httpHandler,
		ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout,
	}
	return &App{Server: server, Logger: logger, pool: pool, telemetry: providers}, nil
}
