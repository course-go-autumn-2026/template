package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/Yolanesso/avito-labs/internal/config"
	"github.com/Yolanesso/avito-labs/internal/database"
	"github.com/Yolanesso/avito-labs/internal/httpserver"
	"github.com/Yolanesso/avito-labs/internal/repository"
	"github.com/Yolanesso/avito-labs/internal/service"
	"github.com/Yolanesso/avito-labs/internal/txmanager"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	connectContext, cancelConnect := context.WithTimeout(
		context.Background(),
		cfg.Database.ConnectTimeout,
	)

	pool, err := database.NewPool(
		connectContext,
		cfg.Database,
	)

	cancelConnect()

	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	defer pool.Close()

	log.Printf("connected to PostgreSQL")

	tripRepository := repository.NewTripRepository(
		pool,
		cfg.Database.QueryTimeout,
	)

	transactionManager := txmanager.New(pool)

	tripService := service.NewTripService(
		tripRepository,
		transactionManager,
	)

	handler := httpserver.NewHandler(
		pool.Ping,
		cfg.Database.QueryTimeout,
		tripService,
	)

	server := httpserver.New(
		cfg.HTTPAddr,
		handler,
	)

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf(
			"trip-service listening on %s",
			cfg.HTTPAddr,
		)

		serverErrors <- server.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server: %v", err)
		}

	case <-signalContext.Done():
		log.Printf("shutdown signal received")

		shutdownContext, cancelShutdown := context.WithTimeout(
			context.Background(),
			cfg.ShutdownTimeout,
		)
		defer cancelShutdown()

		err := server.Shutdown(shutdownContext)
		if err != nil {
			log.Printf(
				"graceful shutdown failed: %v",
				err,
			)
		}
	}

	log.Printf("trip-service stopped")
}
