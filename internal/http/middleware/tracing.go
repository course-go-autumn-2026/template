package middleware

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) func(http.Handler) http.Handler {
	if tracer == nil {
		tracer = otel.GetTracerProvider().Tracer("trip-service")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			spanName := r.Method + " " + r.URL.Path

			ctx, span := tracer.Start(
				ctx,
				spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLPath(r.URL.Path),
				),
			)
			defer span.End()
			defer func() {
				if rec := recover(); rec != nil {
					span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", rec))
					span.RecordError(fmt.Errorf("panic: %v", rec), trace.WithStackTrace(true))
					panic(rec)
				}
			}()

			rw := newStatusResponseWriter(w)
			*r = *r.WithContext(ctx)
			next.ServeHTTP(rw, r)

			route := ""
			if chiCtx := chi.RouteContext(r.Context()); chiCtx != nil {
				route = chiCtx.RoutePattern()
			}
			if route == "" {
				route = r.URL.Path
			}

			span.SetName(fmt.Sprintf("%s %s", r.Method, route))
			span.SetAttributes(
				semconv.HTTPRouteKey.String(route),
				semconv.HTTPResponseStatusCodeKey.Int(rw.statusCode),
			)

			if rw.statusCode >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rw.statusCode))
				span.SetAttributes(attribute.String("error.type", fmt.Sprintf("HTTP %d", rw.statusCode)))
			}
		})
	}
}
