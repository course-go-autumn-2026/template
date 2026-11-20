package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					logger.ErrorContext(
						r.Context(),
						"panic recovered",
						slog.Any("panic", rec),
						slog.String("stack", stack),
					)
					domainerror.Write(w, errors.New("panic recovered"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
