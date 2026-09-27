package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	appconfig "github.com/imrtlDrifter/template/pkg/config"
	"github.com/imrtlDrifter/template/pkg/database"
	"github.com/imrtlDrifter/template/pkg/logger"
)

func Run() {
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	for rootCtx.Err() == nil {
		cfg, setErr := appconfig.Load()
		if setErr != nil {
			log.Printf("load config: %v", setErr)
			os.Exit(1)
		}

		logger.Init(cfg)
		appCtx, cancelApp := context.WithCancel(rootCtx)

		db, err := database.InitDB(appCtx, cfg)
		if err != nil {
			cancelApp()
			logger.ErrorCtx(nil, err, "database_init_failed")
			return
		}

		// Репозитории
		// repos := repositories.NewRepository(db.RedisCluster, db.Postgres)

		// Сервисы
		// svc := service.NewService(repos, cfg)

		// httpHandler := handlers.NewHandler(svc, cfg, db, buildCommit)
		// srv := server.NewServer(cfg, httpHandler.Init(appCtx), configPath)

		stopApp := func() {
			cancelApp()
			db.Close()
		}

		runErrCh := make(chan error, 1)
		go func() {
			if err := srv.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				runErrCh <- err
				return
			}
			runErrCh <- nil
		}()

		log.Println("🚀 Running on port http://localhost" + cfg.Server.Addr + "/")

		needExit := false
		serverFinished := false

		for !needExit {
			select {
			case <-rootCtx.Done():
				logger.InfoCtx(nil, "shutdown_signal_received")
				needExit = true

			case runErr := <-runErrCh:
				serverFinished = true
				needExit = true

				if runErr != nil {
					logger.ErrorCtx(nil, runErr, "server_error")
				}
			}

			if !serverFinished {
				shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)

				shutdownErr := srv.Stop(shutdownCtx)
				cancelShutdown()

				if shutdownErr != nil {
					logger.WarnCtx(nil, shutdownErr, "server_shutdown_failed")
					if closeErr := srv.Close(); closeErr != nil {
						logger.WarnCtx(nil, closeErr, "server_force_close_failed")
					}
				}
				<-runErrCh
			}

			stopApp()

			if needExit {
				return
			}
		}
	}
}
