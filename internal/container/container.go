package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jonboulle/clockwork"
	"go.opentelemetry.io/otel"

	"github.com/George1487/MalyginGeorge/internal/config"
	"github.com/George1487/MalyginGeorge/internal/gateway/push"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	internalHttp "github.com/George1487/MalyginGeorge/internal/http"
	"github.com/George1487/MalyginGeorge/internal/http/middleware"
	tripHttpCreate "github.com/George1487/MalyginGeorge/internal/http/trip/create"
	tripHttpFinish "github.com/George1487/MalyginGeorge/internal/http/trip/finish"
	tripHttpGet "github.com/George1487/MalyginGeorge/internal/http/trip/get"
	positionHttpCreate "github.com/George1487/MalyginGeorge/internal/http/trip/position/create"
	positionHttpGet "github.com/George1487/MalyginGeorge/internal/http/trip/position/get"
	"github.com/George1487/MalyginGeorge/internal/infra/postgres"
	"github.com/George1487/MalyginGeorge/internal/infra/postgres/transaction"
	"github.com/George1487/MalyginGeorge/internal/infra/telemetry"
	"github.com/George1487/MalyginGeorge/internal/repository/idempotency"
	repoTrip "github.com/George1487/MalyginGeorge/internal/repository/trip"
	repoPosition "github.com/George1487/MalyginGeorge/internal/repository/trip/position"
	tripUsecaseCreate "github.com/George1487/MalyginGeorge/internal/usecase/trip/create"
	tripUsecaseFinish "github.com/George1487/MalyginGeorge/internal/usecase/trip/finish"
	tripUsecaseGet "github.com/George1487/MalyginGeorge/internal/usecase/trip/get"
	positionUsecaseCreate "github.com/George1487/MalyginGeorge/internal/usecase/trip/position/create"
	positionUsecaseGet "github.com/George1487/MalyginGeorge/internal/usecase/trip/position/get"
	"github.com/George1487/MalyginGeorge/internal/worker/positions"
)

type Container struct {
	HTTPServer *http.Server
	Pool       *pgxpool.Pool
	Logger     *slog.Logger
	Scaner     *positions.Scaner
	WorkerPool *positions.WorkerPool
	Queue      *positions.Queue
	Shutdown   func(ctx context.Context) error
}

func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.ParsedLogLevel(), cfg.OTelServiceName)

	shutdownOTel, err := telemetry.Init(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("init telemetry: %w", err)
	}

	metrics, err := telemetry.NewMetrics()
	if err != nil {
		_ = shutdownOTel(ctx)
		return nil, fmt.Errorf("init metrics: %w", err)
	}

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		_ = shutdownOTel(ctx)
		return nil, fmt.Errorf("init postgres pool: %w", err)
	}

	txManager := transaction.NewPgxTxManager(pool)
	tripRepo := repoTrip.New(pool)
	positionRepo := repoPosition.New(pool)
	idempotencyStore := idempotency.NewStore(pool)

	queueMu := &sync.Mutex{}
	queue := positions.NewQueue(cfg.PositionQueueSize, queueMu)
	queue.SetObservability(metrics, logger)

	usecaseCreateInstance := tripUsecaseCreate.New(tripRepo, tripRepo, txManager, metrics)
	usecaseGetInstance := tripUsecaseGet.New(tripRepo)
	usecaseFinishInstance := tripUsecaseFinish.New(tripRepo, tripRepo, txManager, metrics)
	usecasePositionCreateInstance := positionUsecaseCreate.New(tripRepo, positionRepo, txManager, metrics, queue)
	usecasePositionGetInstance := positionUsecaseGet.NewUsecase(positionRepo, txManager, tripRepo)

	handlerCreate := tripHttpCreate.New(usecaseCreateInstance)
	handlerGet := tripHttpGet.New(usecaseGetInstance)
	handlerFinish := tripHttpFinish.New(usecaseFinishInstance)
	handlerPositionGet := positionHttpGet.New(usecasePositionGetInstance)
	handlerPositionCreate := positionHttpCreate.New(usecasePositionCreateInstance)

	server := internalHttp.NewServer(handlerCreate, handlerGet, handlerFinish, pool, handlerPositionGet, handlerPositionCreate)

	r := chi.NewRouter()
	tracer := otel.GetTracerProvider().Tracer(cfg.OTelServiceName)

	r.Use(middleware.Recover(logger))
	r.Use(middleware.Tracing(tracer))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.Metrics(metrics, r))

	apiHandler := api.HandlerWithOptions(server, api.ChiServerOptions{
		BaseRouter:       r,
		ErrorHandlerFunc: internalHttp.ErrorHandler,
		Middlewares: []api.MiddlewareFunc{
			middleware.IdempotencyMiddleware(idempotencyStore, txManager, cfg.IdempotencyTTL),
			middleware.RequestTimeout(cfg.DatabaseQueryTimeout),
		},
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiHandler,
		ReadTimeout:       cfg.ShutdownTimeout,
		ReadHeaderTimeout: cfg.ShutdownTimeout,
		WriteTimeout:      cfg.ShutdownTimeout,
		IdleTimeout:       cfg.ShutdownTimeout,
	}

	pushGateway := push.New(cfg.PushHTTPURL, cfg.PushRequestTimeout)
	workerPool := positions.NewWorkerPool(pushGateway, queue, queueMu, metrics, logger)

	clock := clockwork.NewRealClock()
	scanTicker := clock.NewTicker(cfg.PositionCheckInterval)
	scaner := positions.New(tripRepo, scanTicker, metrics, logger, queue)

	var shutdownOnce sync.Once
	shutdown := func(shutdownCtx context.Context) error {
		var errs []error
		shutdownOnce.Do(func() {
			if shutdownOTel != nil {
				if err := shutdownOTel(shutdownCtx); err != nil {
					errs = append(errs, err)
				}
			}
			if pool != nil {
				pool.Close()
			}
		})
		return errors.Join(errs...)
	}

	return &Container{
		HTTPServer: httpServer,
		Pool:       pool,
		Logger:     logger,
		Scaner:     scaner,
		WorkerPool: workerPool,
		Queue:      queue,
		Shutdown:   shutdown,
	}, nil
}
