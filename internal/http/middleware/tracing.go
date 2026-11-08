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
			ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

			initialRoute := resolveRoute(r)
			initialName := fmt.Sprintf("%s %s", r.Method, initialRoute)
			ctx, span := tracer.Start(ctx, initialName,
				trace.WithSpanKind(trace.SpanKindServer),
			)
			defer span.End()

			r = r.WithContext(ctx)
			rw := newResponseWriter(w)

			panicked := true
			defer func() {
				if panicked {
					route := resolveRoute(r)
					span.SetName(fmt.Sprintf("%s %s", r.Method, route))
					span.SetAttributes(
						semconv.HTTPRequestMethodKey.String(r.Method),
						semconv.HTTPResponseStatusCodeKey.Int(http.StatusInternalServerError),
						attribute.String("http.route", route),
						attribute.String("error.type", "internal_error"),
					)
					span.SetStatus(codes.Error, "internal_error")
				}
			}()

			next.ServeHTTP(rw, r)
			panicked = false

			routePattern := resolveRoute(r)
			span.SetName(fmt.Sprintf("%s %s", r.Method, routePattern))
			span.SetAttributes(
				semconv.HTTPRequestMethodKey.String(r.Method),
				semconv.HTTPResponseStatusCodeKey.Int(rw.Status()),
				attribute.String("http.route", routePattern),
			)

			if rw.Status() >= 500 {
				span.SetStatus(codes.Error, http.StatusText(rw.Status()))
				span.SetAttributes(attribute.String("error.type", "internal_error"))
			}
		})
	}
}
