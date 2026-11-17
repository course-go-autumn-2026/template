package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonboulle/clockwork"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	"github.com/freyliee/template/internal/config"
	"github.com/freyliee/template/internal/gateway/push"
	"github.com/freyliee/template/internal/http/health"
	"github.com/freyliee/template/internal/http/middleware"
	"github.com/freyliee/template/internal/http/respond"
	createhandler "github.com/freyliee/template/internal/http/trip/create"
	finishhandler "github.com/freyliee/template/internal/http/trip/finish"
	gethandler "github.com/freyliee/template/internal/http/trip/get"
	listpositionshandler "github.com/freyliee/template/internal/http/trip/listpositions"
	savepositionhandler "github.com/freyliee/template/internal/http/trip/saveposition"
	"github.com/freyliee/template/internal/infra/clock"
	"github.com/freyliee/template/internal/infra/postgres"
	"github.com/freyliee/template/internal/infra/postgres/transaction"
	"github.com/freyliee/template/internal/infra/telemetry"
	repository "github.com/freyliee/template/internal/repository/trip"
	createcase "github.com/freyliee/template/internal/usecase/trip/create"
	finishcase "github.com/freyliee/template/internal/usecase/trip/finish"
	getcase "github.com/freyliee/template/internal/usecase/trip/get"
	listpositionscase "github.com/freyliee/template/internal/usecase/trip/listpositions"
	savepositioncase "github.com/freyliee/template/internal/usecase/trip/saveposition"
	positionworker "github.com/freyliee/template/internal/worker/positions"
)

type App struct {
	Server *http.Server
	Logger *slog.Logger

	pool            *pgxpool.Pool
	telemetry       *telemetry.Telemetry
	active          metric.Registration
	positions       *positionworker.Pool
	push            *push.Client
	positionMetrics metric.Registration
	started         bool
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	var level slog.Level

	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("log level: %w", err)
	}

	logger := telemetry.NewLogger(os.Stdout, level)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) { logger.WarnContext(ctx, "telemetry export failed", "error", err) }))

	tel, err := telemetry.New(ctx, cfg)
	if err != nil {
		return nil, err
	}

	app := &App{Logger: logger, telemetry: tel}
	ready := false

	defer func() {
		if !ready {
			if err := app.Close(ctx); err != nil {
				logger.ErrorContext(ctx, "initialization cleanup failed", "error", err)
			}
		}
	}()

	tracer, err := postgres.NewTracer(tel.Traces, tel.Metrics)
	if err != nil {
		return nil, err
	}

	connectCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseConnectTimeout)

	pool, err := postgres.NewPool(connectCtx, cfg, tracer)
	cancel()

	if err != nil {
		return nil, err
	}

	app.pool = pool

	repo := repository.NewTripRepository(pool)
	tx := transaction.NewTxManager(pool, cfg.DatabaseQueryTimeout, logger)

	business, err := telemetry.NewBusiness(tel.Metrics)
	if err != nil {
		return nil, err
	}

	meter := tel.Metrics.Meter("tripgo.business")

	gauge, err := meter.Int64ObservableGauge("tripgo.trips.active")
	if err != nil {
		return nil, fmt.Errorf("active trips gauge: %w", err)
	}

	app.active, err = meter.RegisterCallback(func(ctx context.Context, observer metric.Observer) error {
		queryCtx, cancel := context.WithTimeout(ctx, cfg.DatabaseQueryTimeout)
		defer cancel()

		count, err := repo.CountActive(queryCtx)
		if err != nil {
			return err
		}

		observer.ObserveInt64(gauge, count)

		return nil
	}, gauge)
	if err != nil {
		return nil, fmt.Errorf("active trips callback: %w", err)
	}

	positionMetrics, err := telemetry.NewPositions(tel.Metrics)
	if err != nil {
		return nil, err
	}

	app.push = push.New(
		cfg.PushHTTPURL,
		cfg.PushRequestTimeout,
		tel.Traces.Tracer("tripgo.push"),
		tel.Propagator,
	)
	app.positions = positionworker.New(
		positionworker.Config{
			Interval:       cfg.PositionCheckInterval,
			StaleAfter:     cfg.PositionStaleAfter,
			Workers:        cfg.PositionWorkers,
			QueueSize:      cfg.PositionQueueSize,
			RequestTimeout: cfg.PushRequestTimeout,
			QueryTimeout:   cfg.DatabaseQueryTimeout,
		},
		repo,
		app.push,
		clockwork.NewRealClock(),
		positionMetrics,
		logger,
		tel.Traces.Tracer("tripgo.positions"),
	)
	app.positionMetrics, err = positionMetrics.ObservePool(app.positions.Snapshot)
	if err != nil {
		return nil, err
	}

	clock := clock.Clock{}
	saveposition := savepositionhandler.New(savepositioncase.New(repo, tx, clock, app.positions, positionMetrics), logger)
	listpositions := listpositionshandler.New(listpositionscase.New(repo), logger)

	create := createhandler.New(createcase.New(repo, tx, clock, business, logger), logger)
	get := gethandler.New(getcase.New(repo), logger)
	finish := finishhandler.New(finishcase.New(repo, tx, clock, business, logger), logger)
	health := health.New(pool, cfg.DatabaseQueryTimeout, logger)

	router := chi.NewRouter()
	router.Get("/health", health.Health)
	router.Get("/ready", health.Ready)
	router.Post("/api/v1/trips", create.ServeHTTP)
	router.Get("/api/v1/trips/{tripId}", get.ServeHTTP)
	router.Post("/api/v1/trips/{tripId}/finish", finish.ServeHTTP)
	router.Post("/api/v1/trips/{tripId}/positions", saveposition.ServeHTTP)
	router.Get("/api/v1/trips/{tripId}/positions", listpositions.ServeHTTP)
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		respond.JSON(w, r, logger, http.StatusNotFound, "application/problem+json", map[string]any{"type": "about:blank", "title": "Not Found", "status": http.StatusNotFound, "code": "not_found"})
	})

	mw, err := middleware.New(logger, tel.Metrics, tel.Traces, tel.Propagator, cfg.DatabaseQueryTimeout)
	if err != nil {
		return nil, err
	}

	app.Server = &http.Server{Addr: cfg.HTTPAddr, Handler: mw.Wrap(router), ReadTimeout: cfg.HTTPReadTimeout, ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout, WriteTimeout: cfg.HTTPWriteTimeout, IdleTimeout: cfg.HTTPIdleTimeout}
	ready = true

	return app, nil
}

func (a *App) Start(ctx context.Context) {
	a.positions.Start(ctx)
	a.started = true
}

func (a *App) Close(ctx context.Context) error {
	var result error

	if a.started {
		result = errors.Join(result, a.positions.Stop(ctx))
	}

	if a.positionMetrics != nil {
		result = errors.Join(result, a.positionMetrics.Unregister())
	}

	if a.push != nil {
		a.push.Close()
	}

	if a.active != nil {
		result = errors.Join(result, a.active.Unregister())
	}

	if a.telemetry != nil {
		if err := a.telemetry.Shutdown(ctx); err != nil {
			a.Logger.WarnContext(ctx, "telemetry shutdown failed", "error", err)
		}
	}

	if a.pool != nil {
		a.pool.Close()
	}

	return result
}
