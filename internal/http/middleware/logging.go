package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newResponseWriter(w)

			panicked := true
			defer func() {
				if panicked && logger != nil {
					durationMs := time.Since(start).Milliseconds()
					route := resolveRoute(r)
					logger.Log(r.Context(), slog.LevelError, "request handled",
						slog.String("method", r.Method),
						slog.String("route", route),
						slog.Int("status", http.StatusInternalServerError),
						slog.Int64("duration_ms", durationMs),
					)
				}
			}()

			next.ServeHTTP(rw, r)
			panicked = false

			durationMs := time.Since(start).Milliseconds()
			route := resolveRoute(r)

			status := rw.Status()
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			}

			if logger != nil {
				logger.Log(r.Context(), level, "request handled",
					slog.String("method", r.Method),
					slog.String("route", route),
					slog.Int("status", status),
					slog.Int64("duration_ms", durationMs),
				)
			}
		})
	}
}
