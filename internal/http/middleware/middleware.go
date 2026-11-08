package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/go-chi/chi/v5"
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
	if routeCtx := chi.RouteContext(r.Context()); routeCtx != nil {
		if pattern := routeCtx.RoutePattern(); pattern != "" {
			return pattern
		}
		// Before dispatch, match against chi's tree without changing the request context.
		matched := chi.NewRouteContext()
		if routeCtx.Routes != nil && routeCtx.Routes.Match(matched, r.Method, r.URL.Path) {
			return matched.RoutePattern()
		}
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
			recorder := &responseWriter{ResponseWriter: w}
			defer func() {
				span.SetName(r.Method + " " + Route(r))
				span.SetAttributes(attribute.String("http.route", Route(r)))
				if value := recover(); value != nil {
					span.SetStatus(codes.Error, "handler panic")
					span.SetAttributes(attribute.String("error.type", "internal_error"), attribute.Int("http.response.status_code", http.StatusInternalServerError))
					// The SDK's deferred span.End records the original exception during unwinding.
					panic(value)
				}
			}()
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
	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10))
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
			completed := false
			defer func() {
				status := recorder.status
				if !completed {
					status = http.StatusInternalServerError
				} else if status == 0 {
					status = http.StatusOK
				}
				attrs = append(attrs, attribute.Int("http.response.status_code", status))
				duration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
			}()
			next.ServeHTTP(recorder, r)
			completed = true
		})
	}, nil
}
