package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

const unknownRoute = "unknown"

func resolveRoute(r *http.Request) string {
	if r == nil {
		return unknownRoute
	}
	if chiCtx := chi.RouteContext(r.Context()); chiCtx != nil {
		if pattern := chiCtx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return matchStaticPattern(r.URL.Path)
}

func matchStaticPattern(path string) string {
	switch {
	case path == "/health":
		return "/health"
	case path == "/ready":
		return "/ready"
	case path == "/api/v1/trips":
		return "/api/v1/trips"
	case strings.HasPrefix(path, "/api/v1/trips/"):
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 4 {
			return "/api/v1/trips/{tripId}"
		}
		if len(parts) == 5 && parts[4] == "finish" {
			return "/api/v1/trips/{tripId}/finish"
		}
		if len(parts) == 5 && parts[4] == "positions" {
			return "/api/v1/trips/{tripId}/positions"
		}
		return unknownRoute
	default:
		return unknownRoute
	}
}
