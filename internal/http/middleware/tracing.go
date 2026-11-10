package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := tracer.Start(r.Context(), r.Method+" "+r.URL.Path)
			defer span.End()
			rec := newStatusRecorder(w)
			r = r.WithContext(ctx)
			next.ServeHTTP(rec, r)
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if pattern := rc.RoutePattern(); pattern != "" {
					span.SetName(r.Method + " " + pattern)
					span.SetAttributes(attribute.String("http.route", pattern))
				}
			}
			span.SetAttributes(attribute.Int("http.response.status_code", rec.Status()))
			if rec.Status() >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rec.Status()))
			}
		})
	}
}
