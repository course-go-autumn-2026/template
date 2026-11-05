package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func Metrics(durationHist metric.Float64Histogram, activeCounter metric.Int64UpDownCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = r.URL.Path
			}

			baseAttrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			}

			if activeCounter != nil {
				activeCounter.Add(r.Context(), 1, metric.WithAttributes(baseAttrs...))
				defer activeCounter.Add(r.Context(), -1, metric.WithAttributes(baseAttrs...))
			}

			next.ServeHTTP(rec, r)

			if durationHist != nil {
				finalRoute := chi.RouteContext(r.Context()).RoutePattern()
				if finalRoute == "" {
					finalRoute = route
				}
				attrs := []attribute.KeyValue{
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", finalRoute),
					attribute.String("http.response.status_code", strconv.Itoa(rec.status)),
				}
				durationHist.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
			}
		})
	}
}
