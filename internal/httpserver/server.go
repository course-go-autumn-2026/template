package httpserver

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	api "github.com/Yolanesso/avito-labs/internal/generated"
)

func New(
	addr string,
	handler api.ServerInterface,
) *http.Server {
	router := chi.NewRouter()

	api.HandlerWithOptions(
		handler,
		api.ChiServerOptions{
			BaseRouter: router,

			ErrorHandlerFunc: func(
				w http.ResponseWriter,
				r *http.Request,
				err error,
			) {
				writeProblem(
					w,
					r,
					http.StatusBadRequest,
					"https://tripgo.example/problems/invalid-request",
					"Invalid request",
					err.Error(),
					"invalid_request",
				)
			},
		},
	)

	return &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
