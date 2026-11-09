package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"

	"github.com/razrabbesezdav-art/template/internal/config"
	"github.com/razrabbesezdav-art/template/internal/database"
	api "github.com/razrabbesezdav-art/template/internal/generated"
	"github.com/razrabbesezdav-art/template/internal/handler"
	"github.com/razrabbesezdav-art/template/internal/repository"
	"github.com/razrabbesezdav-art/template/internal/service"
	"github.com/razrabbesezdav-art/template/internal/transaction"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	pool, err := database.NewPostgres(ctx, cfg)
	if err != nil {
		log.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	txManager := transaction.NewManager(
		transaction.NewPGXBeginner(pool),
	)

	tripsRepo := repository.NewTrips(pool, cfg.DatabaseQueryTimeout)
	historyRepo := repository.NewHistory(pool, cfg.DatabaseQueryTimeout)

	tripService := service.NewTrips(
		txManager,
		tripsRepo,
		historyRepo,
	)

	tripHandler := handler.NewTrips(tripService, pool, cfg.ReadyTimeout)

	router := chi.NewRouter()

	apiHandler := api.HandlerWithOptions(tripHandler, api.ChiServerOptions{
		BaseRouter: router,
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			handler.WriteProblem(w, http.StatusBadRequest, "invalid_request", err.Error())
		},
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apiHandler,
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	serverErr := make(chan error, 1)

	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}

	case <-ctx.Done():
		log.Println("shutdown signal received")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			cfg.ShutdownTimeout,
		)

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
			_ = server.Close()
		}

		cancel()

		if err := <-serverErr; err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server shutdown: %v", err)
		}

		log.Println("server stopped")
	}
}