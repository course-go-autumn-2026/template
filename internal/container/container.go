package container

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/xotapy-main/courseGo/internal/config"
	"github.com/xotapy-main/courseGo/internal/domain/trip"
	api "github.com/xotapy-main/courseGo/internal/generated"
	"github.com/xotapy-main/courseGo/internal/http/domainerror"
	httpmw "github.com/xotapy-main/courseGo/internal/http/middleware"
	httpcreate "github.com/xotapy-main/courseGo/internal/http/trip/create"
	httpfinish "github.com/xotapy-main/courseGo/internal/http/trip/finish"
	httpget "github.com/xotapy-main/courseGo/internal/http/trip/get"
	"github.com/xotapy-main/courseGo/internal/infra/postgres"
	"github.com/xotapy-main/courseGo/internal/infra/postgres/transaction"
	"github.com/xotapy-main/courseGo/internal/infra/telemetry"
	repotrip "github.com/xotapy-main/courseGo/internal/repository/trip"
	uccreate "github.com/xotapy-main/courseGo/internal/usecase/trip/create"
	ucfinish "github.com/xotapy-main/courseGo/internal/usecase/trip/finish"
	ucget "github.com/xotapy-main/courseGo/internal/usecase/trip/get"
)

var _ api.ServerInterface = (*apiAdapter)(nil)

type apiAdapter struct {
	create *httpcreate.Handler
	get    *httpget.Handler
	finish *httpfinish.Handler
	pool   *pgxpool.Pool
}

func (a *apiAdapter) CreateTrip(w http.ResponseWriter, r *http.Request) {
	a.create.ServeHTTP(w, r)
}

func (a *apiAdapter) GetTrip(w http.ResponseWriter, r *http.Request, tripID openapi_types.UUID) {
	a.get.Handle(w, r, tripID)
}

func (a *apiAdapter) FinishTrip(w http.ResponseWriter, r *http.Request, tripID openapi_types.UUID) {
	a.finish.Handle(w, r, tripID)
}

func (a *apiAdapter) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.HealthResponseStatusOk,
	})
}

func (a *apiAdapter) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := a.pool.Ping(ctx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(api.HealthResponse{
			Status: api.HealthResponseStatusUnavailable,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(api.HealthResponse{
		Status: api.HealthResponseStatusOk,
	})
}

type Container struct {
	Router         http.Handler
	Pool           *pgxpool.Pool
	Logger         *slog.Logger
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *sdkmetric.MeterProvider
}

func New(ctx context.Context, cfg *config.Config) (*Container, error) {
	logger := telemetry.InitLogger(cfg.LogLevel)

	tp, err := telemetry.InitTracer(ctx, "trip-service", cfg.OtelEndpoint)
	if err != nil {
		logger.Warn("tracer init warning", slog.String("error", err.Error()))
	}

	metrics, mp, err := telemetry.InitMetrics(ctx, "trip-service", cfg.OtelEndpoint)
	if err != nil {
		logger.Warn("metrics init warning", slog.String("error", err.Error()))
	}

	pool, err := postgres.NewPool(ctx, postgres.Config{
		URL:            cfg.DatabaseURL,
		MaxConns:       cfg.DatabaseMaxConns,
		MinConns:       cfg.DatabaseMinConns,
		MaxConnLife:    cfg.DatabaseMaxConnLife,
		ConnectTimeout: cfg.DatabaseConnectTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("postgres init: %w", err)
	}

	txMgr := transaction.New(pool)
	tripRepo := repotrip.New(pool)

	createUC := uccreate.New(tripRepo, txMgr)
	getUC := ucget.New(tripRepo)
	finishUC := ucfinish.New(tripRepo, txMgr)

	createH := httpcreate.New(createUC, logger, metrics.TripsCreated)
	getH := httpget.New(getUC)
	finishH := httpfinish.New(finishUC, logger, metrics.TripsCompleted)

	adapter := &apiAdapter{
		create: createH,
		get:    getH,
		finish: finishH,
		pool:   pool,
	}

	r := chi.NewRouter()

	// Порядок п. 6.4: Recover -> Tracing -> Logging -> Metrics -> роутер
	r.Use(httpmw.Recover(logger))
	r.Use(func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http.server",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = r.URL.Path
				}
				return fmt.Sprintf("%s %s", r.Method, route)
			}),
		)
	})
	r.Use(httpmw.Logging(logger))
	r.Use(httpmw.Metrics(metrics.HTTPRequestDuration, metrics.HTTPActiveRequests))

	api.HandlerWithOptions(adapter, api.ChiServerOptions{
		BaseRouter: r,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			domainerror.Write(w, r, trip.NewValidationError("param", err.Error()))
		},
	})

	return &Container{
		Router:         r,
		Pool:           pool,
		Logger:         logger,
		TracerProvider: tp,
		MeterProvider:  mp,
	}, nil
}
