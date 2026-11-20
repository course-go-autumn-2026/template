// Package container is the single place that creates and wires dependencies.
package container

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/metric"

	"github.com/n1ckerr0r/avito-labs/internal/config"
	"github.com/n1ckerr0r/avito-labs/internal/gateway/push"
	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	router "github.com/n1ckerr0r/avito-labs/internal/http"
	"github.com/n1ckerr0r/avito-labs/internal/http/middleware"
	createhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/create"
	finishhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/finish"
	gethttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/get"
	positionhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/positions"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres"
	"github.com/n1ckerr0r/avito-labs/internal/infra/postgres/transaction"
	"github.com/n1ckerr0r/avito-labs/internal/infra/telemetry"
	repotrip "github.com/n1ckerr0r/avito-labs/internal/repository/trip"
	createuc "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/create"
	finishuc "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/finish"
	getuc "github.com/n1ckerr0r/avito-labs/internal/usecase/trip/get"
	"github.com/n1ckerr0r/avito-labs/internal/worker/positions"
)

const meterName = "trip-service"

type Container struct {
	Logger *slog.Logger
	Server *http.Server
	Close  func(ctx context.Context) error
}

// New wires every dependency; ctx is the application's lifetime context.
func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger := telemetry.NewLogger(cfg.LogLevel).With("service", cfg.OTELServiceName)
	slog.SetDefault(logger)

	otelProviders, err := telemetry.Setup(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	pool, err := postgres.OpenPool(ctx, cfg, otelProviders.TracerProvider, otelProviders.MeterProvider)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	meter := otelProviders.MeterProvider.Meter(meterName)

	tripMetrics, err := telemetry.NewTripMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("build trip metrics: %w", err)
	}

	activeRequests, err := meter.Int64UpDownCounter("http.server.active_requests",
		metric.WithDescription("Number of in-flight HTTP requests"), metric.WithUnit("{request}"))
	if err != nil {
		return nil, fmt.Errorf("build http.server.active_requests counter: %w", err)
	}

	txManager := transaction.New(pool, cfg.DatabaseQueryTimeout)
	repo := repotrip.New(pool)

	create := createhttp.New(createuc.New(repo, txManager), tripMetrics, logger)
	get := gethttp.New(getuc.New(repo, cfg.DatabaseQueryTimeout))
	finish := finishhttp.New(finishuc.New(repo, txManager), tripMetrics, logger)

	pushClient := push.New(cfg.PushHTTPURL, cfg.PushRequestTimeout)
	worker, err := positions.New(pool, pushClient, logger, meter, otelProviders.TracerProvider.Tracer(meterName), cfg.PositionCheckInterval, cfg.PositionStaleAfter, cfg.PushRequestTimeout, cfg.DatabaseQueryTimeout, cfg.PositionScanBatchSize, cfg.PositionQueueSize, cfg.PositionWorkers)
	if err != nil {
		return nil, fmt.Errorf("build position worker: %w", err)
	}
	positionHandler, err := positionhttp.New(pool, txManager, worker, cfg.DatabaseQueryTimeout, meter, logger)
	if err != nil {
		return nil, fmt.Errorf("build position handler: %w", err)
	}
	worker.Start(context.WithoutCancel(ctx))
	apiHandler := router.New(create, get, finish, pool, cfg.DatabaseQueryTimeout)
	apiHandler.Positions = positionHandler
	mux := api.HandlerWithOptions(apiHandler, api.ChiServerOptions{
		ErrorHandlerFunc: router.ErrorHandler,
		Middlewares:      []api.MiddlewareFunc{middleware.Route(activeRequests)},
	})

	handler := middleware.Recover(logger)(
		otelhttp.NewMiddleware(meterName,
			otelhttp.WithTracerProvider(otelProviders.TracerProvider),
			otelhttp.WithMeterProvider(otelProviders.MeterProvider),
		)(
			middleware.Logging(logger)(mux),
		),
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	return &Container{
		Logger: logger,
		Server: server,
		Close: func(ctx context.Context) error {
			if err := worker.Stop(ctx); err != nil {
				return err
			}
			pushClient.Close()
			pool.Close()
			return otelProviders.Shutdown(ctx)
		},
	}, nil
}
