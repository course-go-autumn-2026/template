package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(meter metric.Meter) func(http.Handler) http.Handler {
	duration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of HTTP server requests."),
	)
	active, _ := meter.Int64UpDownCounter(
		"http.server.active_requests",
		metric.WithUnit("{request}"),
		metric.WithDescription("Number of in-flight HTTP requests."),
	)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := r.URL.Path
			if rc := chi.RouteContext(r.Context()); rc != nil {
				if p := rc.RoutePattern(); p != "" {
					route = p
				}
			}
			attrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			}
			active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			defer active.Add(r.Context(), -1, metric.WithAttributes(attrs...))

			start := time.Now()
			rec := newStatusRecorder(w)
			next.ServeHTTP(rec, r)
			duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", rec.Status()),
			))
		})
	}
}
