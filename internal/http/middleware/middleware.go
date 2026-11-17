package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

	"github.com/freyliee/template/internal/http/domainerror"
	"github.com/freyliee/template/internal/http/respond"
)

type (
	stateKey struct{}

	requestState struct {
		route    string
		response *response
		ctx      context.Context
	}
)

type response struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func (w *response) Header() http.Header { return w.header }

func (w *response) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *response) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}

	return w.body.Write(body)
}

func (w *response) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}

	return w.status
}

type Middleware struct {
	logger *slog.Logger

	duration metric.Float64Histogram
	active   metric.Int64UpDownCounter

	tp         trace.TracerProvider
	propagator propagation.TextMapPropagator
	timeout    time.Duration
}

func New(logger *slog.Logger, mp metric.MeterProvider, tp trace.TracerProvider, propagator propagation.TextMapPropagator, timeout time.Duration) (*Middleware, error) {
	meter := mp.Meter("tripgo.http")

	duration, err := meter.Float64Histogram("http.server.request.duration", metric.WithUnit("s"))
	if err != nil {
		return nil, fmt.Errorf("HTTP duration: %w", err)
	}

	active, err := meter.Int64UpDownCounter("http.server.active_requests")
	if err != nil {
		return nil, fmt.Errorf("active requests: %w", err)
	}

	return &Middleware{logger: logger, duration: duration, active: active, tp: tp, propagator: propagator, timeout: timeout}, nil
}

func (m *Middleware) Wrap(router chi.Router) http.Handler {
	return m.Recover(router, m.Tracing(m.Logging(m.Metrics(router))))
}

func (m *Middleware) Recover(router chi.Router, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := "/unmatched"
		rc := chi.NewRouteContext()
		if router.Match(rc, r.Method, r.URL.Path) {
			route = rc.RoutePattern()
		}

		buffer := &response{header: make(http.Header)}
		state := &requestState{route: route, response: buffer, ctx: r.Context()}
		r = r.WithContext(context.WithValue(r.Context(), stateKey{}, state))
		r.Pattern = route

		defer func() {
			if value := recover(); value != nil {
				m.logger.ErrorContext(state.ctx, "request panic", "panic", fmt.Sprint(value), "stack", string(debug.Stack()))

				buffer.header = make(http.Header)
				buffer.body.Reset()
				buffer.status = 0

				status, problem := domainerror.Problem(errors.New("request panic"))
				instance := r.URL.Path
				problem.Instance = &instance
				respond.JSON(buffer, r.WithContext(state.ctx), m.logger, status, "application/problem+json", problem)
			}

			for key, values := range buffer.header {
				w.Header()[key] = values
			}

			w.WriteHeader(buffer.Status())

			if _, err := w.Write(buffer.body.Bytes()); err != nil {
				m.logger.ErrorContext(state.ctx, "write buffered response", "error", err)
			}
		}()

		next.ServeHTTP(buffer, r)
	})
}

func (m *Middleware) Tracing(next http.Handler) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.Context().Value(stateKey{}).(*requestState)
		state.ctx = r.Context()
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attribute.String("http.route", state.route), attribute.String("http.request.method", r.Method))

		defer func() {
			span.SetAttributes(attribute.Int("http.response.status_code", state.response.Status()))

			if value := recover(); value != nil {
				span.SetAttributes(attribute.String("error.type", "panic"))
				span.SetStatus(codes.Error, "request panic")
				span.RecordError(fmt.Errorf("panic: %v", value))
				w.WriteHeader(http.StatusInternalServerError)
				panic(value)
			}

			if state.response.Status() >= 500 {
				span.SetAttributes(attribute.String("error.type", "internal_error"))
				span.SetStatus(codes.Error, "request failed")
			}
		}()

		next.ServeHTTP(state.response, r)
		w.WriteHeader(state.response.Status())
	})

	return otelhttp.NewHandler(inner, "HTTP", otelhttp.WithTracerProvider(m.tp), otelhttp.WithMeterProvider(noop.NewMeterProvider()), otelhttp.WithPropagators(m.propagator), otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string { return r.Method + " " + r.Pattern }))
}

func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		state := r.Context().Value(stateKey{}).(*requestState)

		defer func() {
			level := slog.LevelInfo

			if state.response.Status() >= 500 {
				level = slog.LevelError
			}

			m.logger.Log(r.Context(), level, "request handled", "method", r.Method, "route", state.route, "status", state.response.Status(), "duration_ms", float64(time.Since(started).Microseconds())/1000)
		}()

		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := r.Context().Value(stateKey{}).(*requestState)
		started := time.Now()
		attrs := []attribute.KeyValue{attribute.String("http.request.method", r.Method), attribute.String("http.route", state.route)}
		m.active.Add(r.Context(), 1, metric.WithAttributes(attrs...))

		defer func() {
			value := recover()
			if value != nil {
				state.response.status = http.StatusInternalServerError
			}

			m.active.Add(r.Context(), -1, metric.WithAttributes(attrs...))
			attrs = append(attrs, attribute.Int("http.response.status_code", state.response.Status()))
			m.duration.Record(r.Context(), time.Since(started).Seconds(), metric.WithAttributes(attrs...))

			if value != nil {
				panic(value)
			}
		}()

		ctx, cancel := context.WithTimeout(r.Context(), m.timeout)
		defer cancel()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
