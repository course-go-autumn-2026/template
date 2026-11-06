package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
)

func Tracing() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = r.URL.Path
				}
				return r.Method + " " + route
			}),
			//nolint:staticcheck
			otelhttp.WithMetricAttributesFn(func(r *http.Request) []attribute.KeyValue {
				route := chi.RouteContext(r.Context()).RoutePattern()
				if route == "" {
					route = "unknown"
				}
				return []attribute.KeyValue{
					attribute.String("http.route", route),
				}
			}),
		)
	}
}
