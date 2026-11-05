package middleware

import (
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			propagator := otel.GetTextMapPropagator()
			ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			initialSpanName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

			ctx, span := tracer.Start(
				ctx,
				initialSpanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(r.Method),
					semconv.URLPathKey.String(r.URL.Path),
				),
			)
			defer span.End()

			wrapped := wrapResponseWriter(w)
			r = r.WithContext(ctx)

			next.ServeHTTP(wrapped, r)

			route := getRoutePattern(r)
			span.SetName(fmt.Sprintf("%s %s", r.Method, route))
			span.SetAttributes(
				attribute.String("http.route", route),
				semconv.HTTPResponseStatusCodeKey.Int(wrapped.Status()),
			)

			if wrapped.Status() >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(wrapped.Status()))
				span.SetAttributes(attribute.String("error.type", http.StatusText(wrapped.Status())))
			}
		})
	}
}
