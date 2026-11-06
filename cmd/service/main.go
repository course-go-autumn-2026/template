package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/toooriii/tori/internal/config"
	"github.com/toooriii/tori/internal/container"
	api "github.com/toooriii/tori/internal/generated"
	mw "github.com/toooriii/tori/internal/http/middleware"
	"github.com/toooriii/tori/internal/infra/telemetry"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	tel, err := telemetry.Init(ctx, cfg.Telemetry, "dev")
	if err != nil {
		return fmt.Errorf("telemetry: %w", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tel.Shutdown(shutCtx); err != nil {
			fmt.Fprintln(os.Stderr, "telemetry shutdown:", err)
		}
	}()

	base := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})
	log := slog.New(telemetry.NewTraceHandler(base))

	c, err := container.New(ctx, cfg, log)
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}
	defer c.Close()

	router := chi.NewRouter()
	router.Use(mw.Recover(log))
	router.Use(mw.Tracing())
	router.Use(mw.Logging(log))

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: api.HandlerWithOptions(
			&server{
				create: c.CreateTrip,
				get:    c.GetTrip,
				finish: c.FinishTrip,
				ping:   c.Ping,
			},
			api.ChiServerOptions{
				BaseRouter:       router,
				ErrorHandlerFunc: paramError,
			},
		),
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		BaseContext:       func(net.Listener) context.Context { return context.Background() },
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("http server started", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	stop()
	log.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())

	time.AfterFunc(cfg.ShutdownTimeout, func() {
		log.Warn("shutdown timeout exceeded, forcing exit")
		os.Exit(1)
	})

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown", "error", err)
	}

	log.Info("stopped")
	return nil
}

type server struct {
	create interface {
		CreateTrip(http.ResponseWriter, *http.Request, api.CreateTripParams)
	}
	get interface {
		GetTrip(http.ResponseWriter, *http.Request, api.TripId)
	}
	finish interface {
		FinishTrip(http.ResponseWriter, *http.Request, api.TripId)
	}
	ping func(ctx context.Context) error
}

func (s *server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.create.CreateTrip(w, r, params)
}

func (s *server) GetTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.get.GetTrip(w, r, id)
}

func (s *server) FinishTrip(w http.ResponseWriter, r *http.Request, id api.TripId) {
	s.finish.FinishTrip(w, r, id)
}

func (s *server) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *server) Ready(w http.ResponseWriter, r *http.Request) {
	if err := s.ping(r.Context()); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"unavailable"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func paramError(w http.ResponseWriter, _ *http.Request, err error) {
	d := err.Error()
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Type:   "https://tripgo.example/problems/invalid-request",
		Title:  "Invalid request",
		Status: http.StatusBadRequest,
		Code:   "invalid_request",
		Detail: &d,
	})
}
