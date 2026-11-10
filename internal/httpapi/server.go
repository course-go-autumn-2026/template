package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/murmyauuu/template/api"
	"github.com/murmyauuu/template/internal/config"
	"github.com/murmyauuu/template/internal/trip"
)

type handler struct {
	pool         *pgxpool.Pool
	trips        *trip.Service
	queryTimeout time.Duration
	logger       *slog.Logger
}

func New(cfg config.HTTP, pool *pgxpool.Pool, trips *trip.Service, queryTimeout time.Duration, logger *slog.Logger) *http.Server {
	h := &handler{pool: pool, trips: trips, queryTimeout: queryTimeout, logger: logger}
	router := api.HandlerWithOptions(h, api.ChiServerOptions{
		BaseRouter: chi.NewRouter(),
		ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			h.problem(r.Context(), w, r.URL.Path, http.StatusBadRequest, "invalid_request", "Invalid request", "Request parameters are invalid")
		},
	})
	return &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

func (h *handler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(r.Context(), w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (h *handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.queryTimeout)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		h.logger.WarnContext(r.Context(), "database readiness check failed", "error", err)
		h.writeJSON(r.Context(), w, http.StatusServiceUnavailable, "application/json", api.HealthResponse{Status: api.Unavailable})
		return
	}
	h.writeJSON(r.Context(), w, http.StatusOK, "application/json", api.HealthResponse{Status: api.Ok})
}

func (h *handler) problem(ctx context.Context, w http.ResponseWriter, instance string, status int, code, title, detail string) {
	h.writeJSON(ctx, w, status, "application/problem+json", api.Problem{
		Type:     "https://tripgo.example/problems/" + strings.ReplaceAll(code, "_", "-"),
		Title:    title,
		Status:   int32(status),
		Code:     code,
		Detail:   &detail,
		Instance: &instance,
	})
}

func (h *handler) writeJSON(ctx context.Context, w http.ResponseWriter, status int, contentType string, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.WarnContext(ctx, "write HTTP response failed", "error", err)
	}
}
