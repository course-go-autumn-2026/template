package httpserver

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", HealthHandler)
	r.Get("/ready", ReadyHandler)

	return r
}
