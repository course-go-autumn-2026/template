package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	appconfig "github.com/imrtlDrifter/template/internal/config"
	"github.com/imrtlDrifter/template/internal/handlers"
	"github.com/imrtlDrifter/template/internal/repositories"
	"github.com/imrtlDrifter/template/internal/service"
	"github.com/imrtlDrifter/template/pkg/database"
	"github.com/imrtlDrifter/template/pkg/logger"
	"github.com/imrtlDrifter/template/pkg/server"
)

func Run(parentCtx context.Context) error {
	rootCtx, stopSignals := signal.NotifyContext(parentCtx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	cfg, err := appconfig.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger.Init(cfg)
	appCtx, cancelApp := context.WithCancel(rootCtx)

	db, err := database.InitDB(appCtx, cfg)
	if err != nil {
		cancelApp()
		return fmt.Errorf("init database: %w", err)
	}
	defer func() {
		cancelApp()
		db.Close()
	}()

	repos := repositories.NewRepository(db.Postgres, cfg.PostgreSQL.QueryTimeout)

	svc, err := service.NewService(repos, db.TxManager)
	if err != nil {
		return fmt.Errorf("init service: %w", err)
	}

	httpHandler := handlers.NewHandler(db, svc)
	srv := server.NewServer(cfg, httpHandler.Init())

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Run()
	}()

	log.Println("🚀 Running on port http://localhost" + cfg.Server.Addr + "/")

	select {
	case runErr := <-runErrCh:
		if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
			return fmt.Errorf("run HTTP server: %w", runErr)
		}
		return nil

	case <-rootCtx.Done():
		logger.InfoCtx(nil, "shutdown_signal_received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.WithoutCancel(rootCtx), cfg.Server.ShutdownTimeout)
	defer cancelShutdown()

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.WarnCtx(nil, err, "server_shutdown_failed")

		if closeErr := srv.Close(); closeErr != nil {
			logger.WarnCtx(nil, closeErr, "server_force_close_failed")
		}

		<-runErrCh
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	runErr := <-runErrCh
	if runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
		return fmt.Errorf("stop HTTP server: %w", runErr)
	}

	return nil
}
