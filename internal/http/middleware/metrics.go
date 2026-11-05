package middleware

import (
	"context"
	"net/http"
	"time"
)

type metricsRecorder interface {
	RecordRequestDuration(ctx context.Context, method, route string, statusCode int, durationSec float64)
	IncActiveRequests(ctx context.Context, method, route string)
	DecActiveRequests(ctx context.Context, method, route string)
}

func Metrics(m metricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			wrapped := wrapResponseWriter(w)

			route := getRoutePattern(r)
			m.IncActiveRequests(r.Context(), r.Method, route)

			defer func() {
				finalRoute := getRoutePattern(r)
				durationSec := time.Since(start).Seconds()

				m.DecActiveRequests(r.Context(), r.Method, route)
				m.RecordRequestDuration(r.Context(), r.Method, finalRoute, wrapped.Status(), durationSec)
			}()

			next.ServeHTTP(wrapped, r)
		})
	}
}
