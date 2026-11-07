package container

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/XrystikLK/template/internal/config"
	domainTrip "github.com/XrystikLK/template/internal/domain/trip"
	api "github.com/XrystikLK/template/internal/generated"
	httpTransport "github.com/XrystikLK/template/internal/http"
	"github.com/XrystikLK/template/internal/http/domainerror"
	"github.com/XrystikLK/template/internal/http/middleware"
	createHandler "github.com/XrystikLK/template/internal/http/trip/create"
	finishHandler "github.com/XrystikLK/template/internal/http/trip/finish"
	getHandler "github.com/XrystikLK/template/internal/http/trip/get"
	"github.com/XrystikLK/template/internal/infra/postgres"
	"github.com/XrystikLK/template/internal/infra/postgres/transaction"
	"github.com/XrystikLK/template/internal/infra/telemetry"
	repoTrip "github.com/XrystikLK/template/internal/repository/trip"
	createUC "github.com/XrystikLK/template/internal/usecase/trip/create"
	finishUC "github.com/XrystikLK/template/internal/usecase/trip/finish"
	getUC "github.com/XrystikLK/template/internal/usecase/trip/get"
)

type Container struct {
	server    *http.Server
	pool      *pgxpool.Pool
	telemetry *telemetry.Telemetry
	logger    *slog.Logger
}

func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.LogLevel, cfg.OtelServiceName)

	tel, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		_ = tel.Shutdown(ctx)
		return nil, err
	}

	txManager := transaction.NewManager(pool)
	tripRepo := repoTrip.New(pool, cfg.DatabaseQueryTimeout)

	createUseCase := createUC.New(tripRepo, txManager, logger, tel.Metrics)
	getUseCase := getUC.New(tripRepo)
	finishUseCase := finishUC.New(tripRepo, txManager, logger, tel.Metrics)

	createH := createHandler.New(createUseCase)
	getH := getHandler.New(getUseCase)
	finishH := finishHandler.New(finishUseCase)

	srv := httpTransport.NewServer(createH, getH, finishH, pool, cfg.DatabaseConnectTimeout)

	r := chi.NewRouter()
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Tracing(tel.Tracer))
	r.Use(middleware.Logging(logger))
	r.Use(middleware.Metrics(tel.Metrics))

	api.HandlerWithOptions(srv, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			domainerror.Write(w, r, &domainTrip.ValidationError{
				Field:   "parameters",
				Message: err.Error(),
			})
		},
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	return &Container{
		server:    httpServer,
		pool:      pool,
		telemetry: tel,
		logger:    logger,
	}, nil
}

func (c *Container) Start() error {
	return c.server.ListenAndServe()
}

func (c *Container) Shutdown(ctx context.Context) error {
	defer c.pool.Close()
	defer func() {
		_ = c.telemetry.Shutdown(ctx)
	}()
	return c.server.Shutdown(ctx)
}

func (c *Container) Addr() string {
	return c.server.Addr
}

func (c *Container) Logger() *slog.Logger {
	return c.logger
}
