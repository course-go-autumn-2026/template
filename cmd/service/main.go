package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/ggwpfu9/courseAvito/internal/container"
	"github.com/ggwpfu9/courseAvito/internal/http/middleware"
	createH "github.com/ggwpfu9/courseAvito/internal/http/trip/create"
	finishH "github.com/ggwpfu9/courseAvito/internal/http/trip/finish"
	getH "github.com/ggwpfu9/courseAvito/internal/http/trip/get"
	"github.com/ggwpfu9/courseAvito/internal/infra/telemetry"
)

func main() {
	_ = godotenv.Load()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx := context.Background()

	shutdown, err := telemetry.Init(ctx, "trip-service")
	if err != nil {
		logger.Error("failed to init telemetry", "error", err)
		os.Exit(1)
	}
	defer func() { _ = shutdown(ctx) }()

	dbURL := os.Getenv("DATABASE_URL")
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		logger.Error("failed to parse db url", "error", err)
		os.Exit(1)
	}
	cfg.ConnConfig.Tracer = otelpgx.NewTracer()
	
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		logger.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	c := container.New(pool)

	r := chi.NewRouter()
	r.Use(middleware.Logging)

	r.Post("/api/v1/trips", createH.New(c.CreateTrip).ServeHTTP)
	r.Get("/api/v1/trips/{tripId}", getH.New(c.GetTrip).ServeHTTP)
	r.Post("/api/v1/trips/{tripId}/finish", finishH.New(c.FinishTrip).ServeHTTP)
	
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	handler := otelhttp.NewHandler(r, "http-server")

	logger.Info("Starting server on :8080")
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		logger.Error("server error", "error", err)
	}
}