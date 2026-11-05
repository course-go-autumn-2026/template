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
			wrapped := wrapResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			durationMs := time.Since(start).Milliseconds()
			route := getRoutePattern(r)
			status := wrapped.Status()

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", status),
				slog.Int64("duration_ms", durationMs),
			}

			if status >= http.StatusInternalServerError {
				logger.LogAttrs(r.Context(), slog.LevelError, "request handled", attrs...)
			} else {
				logger.LogAttrs(r.Context(), slog.LevelInfo, "request handled", attrs...)
			}
		})
	}
}
