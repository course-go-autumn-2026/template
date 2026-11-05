package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"
)

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		spanCtx := trace.SpanContextFromContext(r.Context())
		traceID := "none"
		if spanCtx.IsValid() {
			traceID = spanCtx.TraceID().String()
		}

		level := slog.LevelInfo
		if ww.Status() >= 500 {
			level = slog.LevelError
		}

		slog.Log(r.Context(), level, "request handled",
			"method", r.Method,
			"route", r.URL.Path,
			"status", ww.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"trace_id", traceID,
		)
	})
}