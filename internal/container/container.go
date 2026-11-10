package container

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/config"
	httpx "github.com/murmyauuu/template/internal/http"
	"github.com/murmyauuu/template/internal/http/middleware"
	tripcreate "github.com/murmyauuu/template/internal/http/trip/create"
	tripfinish "github.com/murmyauuu/template/internal/http/trip/finish"
	tripget "github.com/murmyauuu/template/internal/http/trip/get"
	"github.com/murmyauuu/template/internal/infra/postgres"
	"github.com/murmyauuu/template/internal/infra/postgres/transaction"
	"github.com/murmyauuu/template/internal/infra/telemetry"
	"github.com/murmyauuu/template/internal/repository/trip"
	uccreate "github.com/murmyauuu/template/internal/usecase/trip/create"
	ucfinish "github.com/murmyauuu/template/internal/usecase/trip/finish"
	ucget "github.com/murmyauuu/template/internal/usecase/trip/get"
)

type Container struct {
	Server   *http.Server
	Logger   *slog.Logger
	Shutdown func(context.Context) error
}

func New(ctx context.Context, cfg config.Config) (*Container, error) {
	logger, shutdown, err := telemetry.Setup(ctx, telemetry.Config{
		ServiceName: "trip-service",
		Version:     "dev",
		Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		Protocol:    os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
		Sampler:     os.Getenv("OTEL_TRACES_SAMPLER"),
		LogLevel:    cfg.LogLevel,
	})
	if err != nil {
		return nil, fmt.Errorf("setup telemetry: %w", err)
	}

	pool, err := postgres.Open(ctx, cfg.Database)
	if err != nil {
		_ = shutdown(ctx)
		return nil, fmt.Errorf("open database: %w", err)
	}

	tx := transaction.NewTxManager(pool, cfg.Database.QueryTimeout)
	repo := trip.NewRepository(pool, cfg.Database.QueryTimeout)
	history := trip.NewHistoryRepository(pool, cfg.Database.QueryTimeout)
	idempotency := trip.NewIdempotencyRepository(pool, cfg.Database.QueryTimeout)

	createUC := uccreate.New(repo, tx, history, idempotency, cfg.IdempotencyTTL, logger)
	getUC := ucget.New(repo)
	finishUC := ucfinish.New(repo, tx, history, logger)

	createHandler := &tripcreate.Handler{Usecase: createUC, Logger: logger}
	getHandler := &tripget.Handler{Usecase: getUC, Logger: logger}
	finishHandler := &tripfinish.Handler{Usecase: finishUC, Logger: logger}

	composite := &server{
		create:       createHandler,
		get:          getHandler,
		finish:       finishHandler,
		pool:         pool,
		queryTimeout: cfg.Database.QueryTimeout,
		logger:       logger,
	}

	tracer := otelTracer()
	meter := otelMeter()

	router := chi.NewRouter()
	router.Use(middleware.Recover(logger))
	router.Use(middleware.Tracing(tracer))
	router.Use(middleware.Logging(logger))
	router.Use(middleware.Metrics(meter))

	api.HandlerFromMux(composite, router)

	srv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	c := &Container{Server: srv, Logger: logger, Shutdown: func(ctx context.Context) error {
		var err error
		err = errors.Join(err, srv.Shutdown(ctx))
		pool.Close()
		err = errors.Join(err, shutdown(ctx))
		return err
	}}
	return c, nil
}

type server struct {
	create       *tripcreate.Handler
	get          *tripget.Handler
	finish       *tripfinish.Handler
	pool         *pgxpool.Pool
	queryTimeout time.Duration
	logger       *slog.Logger
}

func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.create.CreateTrip(w, r, params)
}

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.get.GetTrip(w, r, tripID)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.finish.FinishTrip(w, r, tripID)
}

func (s *server) Health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.queryTimeout)
	defer cancel()
	if err := s.pool.Ping(ctx); err != nil {
		s.logger.WarnContext(r.Context(), "database readiness check failed", "error", err)
		httpx.WriteJSON(w, http.StatusServiceUnavailable, "application/json", api.HealthResponse{Status: api.Unavailable})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func otelTracer() trace.Tracer {
	return otel.Tracer("trip-service")
}

func otelMeter() metric.Meter {
	return otel.Meter("trip-service")
}
