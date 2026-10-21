package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	api "github.com/whxtelyy/template/internal/generated"
)

type Wrapper func(http.Handler) http.Handler

func Route(r *http.Request) string {
	path := r.URL.Path
	if path == "/health" || path == "/ready" || path == "/api/v1/trips" {
		return path
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "trips" {
		return "/api/v1/trips/{tripId}"
	}
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "v1" && parts[2] == "trips" && parts[4] == "finish" {
		return "/api/v1/trips/{tripId}/finish"
	}
	return "unmatched"
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *responseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

func Recover(logger *slog.Logger) Wrapper {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			defer func() {
				if value := recover(); value != nil {
					logger.ErrorContext(r.Context(), "request panic", "panic", value, "stack", string(debug.Stack()))
					logger.ErrorContext(r.Context(), "request handled", "method", r.Method, "route", Route(r), "status", http.StatusInternalServerError, "duration_ms", time.Since(start).Milliseconds())
					detail, instance := "Internal server error", r.URL.Path
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(api.Problem{Type: "https://tripgo.example/problems/internal-error", Title: "Internal Server Error", Status: 500, Detail: &detail, Instance: &instance, Code: "internal_error"})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Tracing(provider trace.TracerProvider) Wrapper {
	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			span := trace.SpanFromContext(r.Context())
			span.SetAttributes(attribute.String("http.route", Route(r)))
			recorder := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			if recorder.status >= 500 {
				span.SetStatus(codes.Error, "server error")
				span.SetAttributes(attribute.String("error.type", "internal_error"))
			}
		})
		return otelhttp.NewHandler(inner, "trip-service", otelhttp.WithTracerProvider(provider),
			otelhttp.WithMeterProvider(noop.NewMeterProvider()),
			otelhttp.WithPropagators(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})),
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method + " " + Route(r) }))
	}
}

func Logging(logger *slog.Logger) Wrapper {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			recorder := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			args := []any{"method", r.Method, "route", Route(r), "status", status, "duration_ms", time.Since(start).Milliseconds()}
			if status >= 500 {
				logger.ErrorContext(r.Context(), "request handled", args...)
			} else {
				logger.InfoContext(r.Context(), "request handled", args...)
			}
		})
	}
}

func Metrics(meter metric.Meter) (Wrapper, error) {
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, err
	}
	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, err
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := Route(r)
			attrs := make([]attribute.KeyValue, 0, 3)
			attrs = append(attrs, attribute.String("http.request.method", r.Method), attribute.String("http.route", route))
			active.Add(r.Context(), 1, metric.WithAttributes(attrs...))
			defer active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
			start := time.Now()
			recorder := &responseWriter{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			attrs = append(attrs, attribute.Int("http.response.status_code", status))
			duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
		})
	}, nil
}
