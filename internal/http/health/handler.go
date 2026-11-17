package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/freyliee/template/internal/http/respond"
)

type (
	Handler struct {
		database database
		timeout  time.Duration
		logger   *slog.Logger
	}
)

func New(database database, timeout time.Duration, logger *slog.Logger) *Handler {
	return &Handler{database: database, timeout: timeout, logger: logger}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, r, h.logger, http.StatusOK, "application/json", map[string]string{"status": "ok"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
	defer cancel()

	status, value := http.StatusOK, "ok"

	if err := h.database.Ping(ctx); err != nil {
		status, value = http.StatusServiceUnavailable, "unavailable"
	}

	respond.JSON(w, r, h.logger, status, "application/json", map[string]string{"status": value})
}
