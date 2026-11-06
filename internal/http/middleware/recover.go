package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5"

	api "github.com/toooriii/tori/internal/generated"
)

func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if rec == http.ErrAbortHandler {
						panic(rec)
					}
					route := chi.RouteContext(r.Context()).RoutePattern()
					if route == "" {
						route = r.URL.Path
					}
					log.ErrorContext(r.Context(), "panic in handler",
						"panic", rec,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"route", route,
					)
					p := api.Problem{
						Type:   "https://tripgo.example/problems/internal-error",
						Title:  "Internal Server Error",
						Status: http.StatusInternalServerError,
						Code:   "internal_error",
					}
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(p)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
