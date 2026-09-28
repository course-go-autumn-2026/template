package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/makkssik/template/internal/config"
	api "github.com/makkssik/template/internal/generated"
)

func NewServer(cfg config.Config, implementation *Implementation) *http.Server {
	router := chi.NewRouter()
	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, r, http.StatusNotFound, "not_found", "Not found", "Route was not found")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		for _, method := range []string{
			http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodConnect, http.MethodTrace,
		} {
			if router.Match(chi.NewRouteContext(), method, r.URL.EscapedPath()) {
				w.Header().Add("Allow", method)
			}
		}
		writeProblem(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed", "Method is not allowed for this route")
	})

	return &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: api.HandlerWithOptions(implementation, api.ChiServerOptions{
			BaseRouter:       router,
			ErrorHandlerFunc: writeRequestError,
		}),
		ReadTimeout:       cfg.HTTPReadTimeout,
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}
}

func writeRequestError(w http.ResponseWriter, r *http.Request, _ error) {
	writeProblem(w, r, http.StatusBadRequest, "invalid_request", "Invalid request", "Request parameters are invalid")
}
