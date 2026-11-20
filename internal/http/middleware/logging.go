package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newStatusResponseWriter(w)

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			route := ""
			if chiCtx := chi.RouteContext(r.Context()); chiCtx != nil {
				route = chiCtx.RoutePattern()
			}
			if route == "" {
				route = r.URL.Path
			}

			level := slog.LevelInfo
			if rw.statusCode >= 500 {
				level = slog.LevelError
			}

			logger.Log(
				r.Context(),
				level,
				"request handled",
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", rw.statusCode),
				slog.Int64("duration_ms", duration.Milliseconds()),
			)
		})
	}
}
