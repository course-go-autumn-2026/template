package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/XrystikLK/template/internal/http/domainerror"
)

func Recover(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if p := recover(); p != nil {
					stack := string(debug.Stack())
					if logger != nil {
						logger.ErrorContext(r.Context(), "panic recovered",
							slog.Any("panic", p),
							slog.String("stack", stack),
						)
					}
					domainerror.Write(w, r, errors.New("internal server error"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
