package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/makkssik/template/internal/app"
	"github.com/makkssik/template/internal/config"
	historypg "github.com/makkssik/template/internal/domain/trip_status_history/repositories/pg"
	tripspg "github.com/makkssik/template/internal/domain/trips/repositories/pg"
	"github.com/makkssik/template/internal/postgres"
	checkupservice "github.com/makkssik/template/internal/service/checkup"
	historyservice "github.com/makkssik/template/internal/service/trip_status_history"
	tripsservice "github.com/makkssik/template/internal/service/trips"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}

	historyService := historyservice.NewService(historyservice.ServiceCfg{
		StatusHistoryRepository: historypg.NewPostgresqlRepository(pool, cfg.DatabaseQueryTimeout),
	})
	tripsService := tripsservice.NewService(tripsservice.ServiceCfg{
		TripsRepository:         tripspg.NewPostgresqlRepository(pool, cfg.DatabaseQueryTimeout),
		StatusHistoryRepository: historyService,
		TxManager:               postgres.NewTxManager(pool, cfg.DatabaseQueryTimeout),
	})

	checkupService := checkupservice.NewService(checkupservice.ServiceCfg{
		Database:     pool,
		QueryTimeout: cfg.DatabaseQueryTimeout,
	})
	implementation := app.New(app.ImplementationConfig{
		TripsService:   tripsService,
		CheckupService: checkupService,
	})
	server := app.NewServer(cfg, implementation)
	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- server.ListenAndServe()
	}()
	log.Printf("starting HTTP server on %s", cfg.HTTPAddr)

	var serverErr error
	select {
	case <-ctx.Done():
		log.Print("stopping service")
	case serverErr = <-serverErrors:
	}
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown HTTP server, forcing exit: %v", err)
	}

	poolClosed := make(chan struct{})
	go func() {
		pool.Close()
		close(poolClosed)
	}()

	select {
	case <-poolClosed:
	case <-shutdownCtx.Done():
		log.Fatalf("shutdown timeout, forcing exit: %v", shutdownCtx.Err())
	}

	if serverErr != nil && !errors.Is(serverErr, http.ErrServerClosed) {
		log.Fatalf("serve HTTP: %v", serverErr)
	}
	log.Print("service stopped")
}
