package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/Godofdeath-dev/avitoLabs/internal/config"
	api "github.com/Godofdeath-dev/avitoLabs/internal/generated"
	"github.com/Godofdeath-dev/avitoLabs/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// хелпер для ошибок
func writeProblem(w http.ResponseWriter, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(api.Problem{
		Status: int32(status),
		Code:   code,
		Title:  title,
		Detail: &detail,
	})
}

// Server хранит общие зависимости приложения (пул БД, сервисы)
// и реализует интерфейс api.ServerInterface.
type Server struct {
	pool      *pgxpool.Pool
	repo      *repository.TripRepository
	txManager repository.TxManager
}

// CreateTripPosition implements [api.ServerInterface].
func (s *Server) CreateTripPosition(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// ListTripPositions implements [api.ServerInterface].
func (s *Server) ListTripPositions(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

// Health обрабатывает запрос GET /health (Liveness-проба).
// Отвечает 200 OK до тех пор, пока процесс запущен и способен отвечать
// В противном случае клиент не дождётся ответ
// БД не проверяет
func (s *Server) Health(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK) //200

	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

// Ready обрабатывает запрос GET /ready
// проверяет жива ли бд
func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	// Если пул не инициализирован или метод Ping вернул ошибку:
	if s.pool == nil || s.pool.Ping(ctx) != nil {
		w.WriteHeader(http.StatusServiceUnavailable) // 503
		_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Unavailable})
		return
	}
	w.WriteHeader(http.StatusOK) // 200
	_ = json.NewEncoder(w).Encode(api.HealthResponse{Status: api.Ok})
}

// POST /api/v1/trips
func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	var body api.CreateTripJSONRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeProblem(w, http.StatusBadRequest, "bad_request", "Invalid JSON", err.Error())
		return
	}

	if body.Price <= 0 {
		writeProblem(w, http.StatusBadRequest, "bad_request", "Invalid price", "Price must be greater than zero")
		return
	}
	if body.StartPoint.Latitude < -90 || body.StartPoint.Latitude > 90 ||
		body.StartPoint.Longitude < -180 || body.StartPoint.Longitude > 180 ||
		body.EndPoint.Latitude < -90 || body.EndPoint.Latitude > 90 ||
		body.EndPoint.Longitude < -180 || body.EndPoint.Longitude > 180 {
		writeProblem(w, http.StatusBadRequest, "bad_request", "Invalid coordinates", "Coordinates are out of range")
		return
	}

	newTrip := api.Trip{
		Id:         uuid.New(),
		UserId:     body.UserId,
		DriverId:   body.DriverId,
		Price:      body.Price,
		Status:     api.Active,
		StartPoint: body.StartPoint,
		EndPoint:   body.EndPoint,
		StartedAt:  time.Now().UTC(),
	}

	// Выполняем вставку в trips и в trip_status_history атомарно через TxManager
	err := s.txManager.Do(r.Context(), func(ctx context.Context) error {
		if err := s.repo.InsertTrip(ctx, newTrip); err != nil {
			return err
		}
		return s.repo.AddStatusHistory(ctx, newTrip.Id, nil, newTrip.Status, newTrip.StartedAt)
	})
	if err != nil {
		slog.Error("failed to create trip", "error", err)
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal Error", "Failed to save trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newTrip)
}

// GET /api/v1/trips/{tripId}
func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	trip, err := s.repo.GetTripByID(r.Context(), tripId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		slog.Error("failed to get trip", "error", err)
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal Error", "Failed to get trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}

// POST /api/v1/trips/{tripId}/finish
func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	now := time.Now().UTC()
	var trip *api.Trip

	err := s.txManager.Do(r.Context(), func(ctx context.Context) error {
		var err error
		trip, err = s.repo.UpdateTripCompleted(ctx, tripId, now)
		if err != nil {
			return err
		}
		fromStatus := string(api.Active)
		return s.repo.AddStatusHistory(ctx, tripId, &fromStatus, api.Completed, now)
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeProblem(w, http.StatusNotFound, "trip_not_found", "Not Found", "Trip not found")
			return
		}
		if errors.Is(err, repository.ErrAlreadyFinished) {
			writeProblem(w, http.StatusConflict, "trip_already_completed", "Conflict", "Trip is already completed")
			return
		}
		slog.Error("failed to finish trip", "error", err)
		writeProblem(w, http.StatusInternalServerError, "internal_error", "Internal Error", "Failed to finish trip")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(trip)
}
func main() {
	// 1. Читаем конфигурацию из переменных окружения
	cfg, err := config.Load()
	if err != nil {
		// Если обязательные переменные (HTTP_ADDR, DATABASE_URL) не переданы - выходим
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Настраиваем глобальный логгер, чтобы он писал логи в формате JSON в stdout
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 2. Инициализируем пул соединений с PostgreSQL
	ctx := context.Background()
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL) // парсим конфиг
	if err != nil {
		slog.Error("failed to parse database url", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = cfg.DatabaseMaxConns
	poolCfg.MinConns = cfg.DatabaseMinConns
	poolCfg.MaxConnLifetime = cfg.DatabaseMaxConnLifetime
	poolCfg.ConnConfig.ConnectTimeout = cfg.DatabaseConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg) // подключаем бдшку
	if err != nil {
		slog.Error("failed to create database pool:", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	poolCtx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseConnectTimeout)
	defer cancel()
	if err := pool.Ping(poolCtx); err != nil {
		slog.Error("failed to ping database:", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to postgres successfully")

	// 3. Создаем роутер
	r := chi.NewRouter()
	// добавляет индификаторы
	r.Use(middleware.RequestID)
	// Перехватывает паники
	r.Use(middleware.Recoverer)
	// репозиторий для сервера
	repo := repository.NewTripRepository(pool)
	txManager := repository.NewTxManager(pool)

	serverImpl := &Server{
		pool:      pool,
		repo:      repo,
		txManager: txManager,
	}

	// Регистрируем маршруты OpenAPI в роутере chi
	api.HandlerFromMux(serverImpl, r)

	// 4. запускаем сервер http сервер
	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 2 * time.Second,  // Время на вычитывание HTTP-заголовков
		ReadTimeout:       5 * time.Second,  // Время на вычитывание всего тела запроса
		WriteTimeout:      5 * time.Second,  // Время на запись ответа клиенту
		IdleTimeout:       30 * time.Second, // Время жизни неактивного keep-alive соединения
	}

	serverErrors := make(chan error, 1)

	// Запускаем слушание порта в фоновой горутине, чтобы не блокировать выполнение main
	go func() {
		slog.Info("starting HTTP server", "addr", cfg.HTTPAddr)
		// ListenAndServe блокирует поток
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// 5. Ожидаем сигналы операционной системы для Graceful Shutdown

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors: // сервер упал
		slog.Error("server failed to start", "error", err)
		os.Exit(1)

	case sig := <-shutdownSignals: // shutdown

		slog.Info("shutdown signal received", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("graceful shutdown failed, forcing close", "error", err)
			_ = httpServer.Close()
		}
		slog.Info("server stopped gracefully")
	}
}
