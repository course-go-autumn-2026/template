package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func getRoutePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx != nil {
		pattern := rctx.RoutePattern()
		if pattern != "" {
			return pattern
		}
	}
	return r.URL.Path
}
