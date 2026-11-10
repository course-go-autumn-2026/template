package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/murmyauuu/template/api"
	httpx "github.com/murmyauuu/template/internal/http"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					if logger != nil {
						logger.ErrorContext(r.Context(), "panic recovered", "panic", p, "stack", string(debug.Stack()))
					}
					body := api.Problem{
						Type:   "https://tripgo.example/problems/internal-error",
						Title:  "Internal server error",
						Status: 500,
						Code:   "internal_error",
					}
					detail := "Unable to process the request"
					body.Detail = &detail
					httpx.WriteProblem(w, 500, body)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
