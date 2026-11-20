package middleware

import (
	"context"
	"net/http"
	"time"

	api "github.com/George1487/MalyginGeorge/internal/generated"
)

func RequestTimeout(timeout time.Duration) api.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			ctx, cancel := context.WithTimeout(request.Context(), timeout)
			defer cancel()

			next.ServeHTTP(w, request.WithContext(ctx))
		})
	}
}
