package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type ServerMetrics interface {
	RecordRequestDuration(ctx context.Context, durationSec float64, method, route, statusCode string)
	AddActiveRequests(ctx context.Context, delta int64, method, route string)
}

func Metrics(m ServerMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if m == nil {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			route := resolveRoute(r)

			m.AddActiveRequests(r.Context(), 1, r.Method, route)
			defer m.AddActiveRequests(r.Context(), -1, r.Method, route)

			rw := newResponseWriter(w)
			panicked := true
			defer func() {
				if panicked {
					finalRoute := resolveRoute(r)
					durationSec := time.Since(start).Seconds()
					m.RecordRequestDuration(r.Context(), durationSec, r.Method, finalRoute, "500")
				}
			}()

			next.ServeHTTP(rw, r)
			panicked = false

			finalRoute := resolveRoute(r)
			durationSec := time.Since(start).Seconds()
			m.RecordRequestDuration(r.Context(), durationSec, r.Method, finalRoute, strconv.Itoa(rw.Status()))
		})
	}
}
