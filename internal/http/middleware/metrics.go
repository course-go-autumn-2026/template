package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

func Metrics(router chi.Routes, durationHist metric.Float64Histogram, activeCounter metric.Int64UpDownCounter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			// Определяем шаблон маршрута заранее через chi.Routes.Match, чтобы в active_requests не попал UUID
			tctx := chi.NewRouteContext()
			var route string
			if router != nil && router.Match(tctx, r.Method, r.URL.Path) {
				route = tctx.RoutePattern()
			}
			if route == "" {
				route = "unknown"
			}

			activeAttrs := []attribute.KeyValue{
				attribute.String("http.request.method", r.Method),
				attribute.String("http.route", route),
			}

			if activeCounter != nil {
				activeCounter.Add(r.Context(), 1, metric.WithAttributes(activeAttrs...))
				defer activeCounter.Add(r.Context(), -1, metric.WithAttributes(activeAttrs...))
			}

			next.ServeHTTP(rec, r)

			// После next.ServeHTTP берем точный шаблон из контекста chi
			if afterRoute := chi.RouteContext(r.Context()).RoutePattern(); afterRoute != "" {
				route = afterRoute
			}

			// Переименовываем спан в конце запроса: "{method} {http.route}"
			span := trace.SpanFromContext(r.Context())
			if span.IsRecording() {
				span.SetName(fmt.Sprintf("%s %s", r.Method, route))
				span.SetAttributes(semconv.HTTPRoute(route))
				if rec.status >= http.StatusInternalServerError {
					span.SetStatus(codes.Error, http.StatusText(rec.status))
					span.SetAttributes(attribute.String("error.type", strconv.Itoa(rec.status)))
				}
			}

			if durationHist != nil {
				attrs := []attribute.KeyValue{
					attribute.String("http.request.method", r.Method),
					attribute.String("http.route", route),
					attribute.String("http.response.status_code", strconv.Itoa(rec.status)),
				}
				durationHist.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
			}
		})
	}
}
