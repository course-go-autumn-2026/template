package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type metricsRecorder interface {
	RecordHTTPRequest(ctx context.Context, method, route string, statusCode int, duration time.Duration)
	AddActiveRequest(ctx context.Context, method, route string, delta int64)
}

func Metrics(m metricsRecorder, routes chi.Routes) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := newStatusResponseWriter(w)

			route := ""
			if routes != nil {
				route = routes.Find(chi.NewRouteContext(), r.Method, r.URL.Path)
			}
			if route == "" {
				route = "notFound"
			}

			if m != nil {
				m.AddActiveRequest(r.Context(), r.Method, route, 1)
				defer func() {
					m.AddActiveRequest(r.Context(), r.Method, route, -1)
				}()
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			if m != nil {
				m.RecordHTTPRequest(r.Context(), r.Method, route, rw.statusCode, duration)
			}
		})
	}
}
