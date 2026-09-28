package httpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/poiuyttq/tripgo/internal/repository"
	"github.com/poiuyttq/tripgo/internal/service"
	"github.com/poiuyttq/tripgo/internal/txmanager"
)

func NewRouter(pool *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	tripRepo := repository.NewTripRepository(pool)
	tripHistoryRepo := repository.NewTripStatusHistoryRepository()
	txMgr := txmanager.New(pool)
	tripService := service.NewTripService(tripRepo, tripHistoryRepo, txMgr)

	r.Get("/health", HealthHandler)
	r.Get("/ready", ReadyHandler(pool))
	r.Post("/api/v1/trips", CreateTripHandler(tripService))
	r.Get("/api/v1/trips/{tripId}", GetTripHandler(tripService))
	r.Post("/api/v1/trips/{tripId}/finish", FinishTripHandler(tripService))

	return r
}
