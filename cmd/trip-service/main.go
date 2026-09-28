package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hil-sys/TripGo-autumn/internal/config"
	"github.com/Hil-sys/TripGo-autumn/internal/repository"
	"github.com/Hil-sys/TripGo-autumn/internal/handler"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/jackc/pgx/v5/pgxpool"

	api "github.com/Hil-sys/TripGo-autumn/internal/api"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Некорректный DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = cfg.DatabaseMaxConns
	poolConfig.MinConns = cfg.DatabaseMinConns

	initCtx, initCancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer initCancel()

	pool, err := pgxpool.NewWithConfig(initCtx, poolConfig)
	if err != nil {
		log.Fatalf("Не удалось создать пул: %v", err)
	}

	if err := pool.Ping(initCtx); err != nil {
		log.Fatalf("БД недоступна: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL (Ping OK)")

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Инициализируем слои данных и бизнес-логики
	txManager := repository.NewTxManager(pool)
	tripRepo := repository.NewTripRepository(txManager)
	
	// Создаем наш TripHandler
	tripHandler := handler.NewTripHandler(tripRepo, txManager)

	// связываем сгенерированные пути с нашей структурой
	r.Mount("/", api.Handler(tripHandler))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       15 * time.Second,
	}

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Запуск сгенерированного HTTP-сервера на %s...", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	sig := <-shutdownSignals
	log.Printf("Получен сигнал %v. Начинаем Graceful Shutdown...", sig)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Принудительная остановка: %v", err)
	} else {
		log.Println("HTTP-сервер успешно остановлен.")
	}

	log.Println("Закрываем пул подключений к PostgreSQL...")
	pool.Close()
	log.Println("Сервис полностью завершил работу.")
}
