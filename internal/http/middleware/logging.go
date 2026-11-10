package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/go-chi/chi/v5"
)

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newStatusRecorder(w)
			next.ServeHTTP(rec, r)
			duration := time.Since(start)
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					route = p
				}
			}
			level := slog.LevelInfo
			if rec.Status() >= 500 {
				level = slog.LevelError
			}
			attrs := []slog.Attr{
				slog.String("service", "trip-service"),
				slog.String("method", r.Method),
				slog.String("route", route),
				slog.Int("status", rec.Status()),
				slog.Int64("duration_ms", duration.Milliseconds()),
			}
			span := trace.SpanFromContext(r.Context())
			sc := span.SpanContext()
			if sc.HasTraceID() {
				attrs = append(attrs, slog.String("trace_id", sc.TraceID().String()))
			}
			if sc.HasSpanID() {
				attrs = append(attrs, slog.String("span_id", sc.SpanID().String()))
			}
			if logger != nil {
				logger.LogAttrs(r.Context(), level, "request handled", attrs...)
			}
		})
	}
}
