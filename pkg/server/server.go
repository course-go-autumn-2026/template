package server

import (
	"context"
	"net/http"

	"github.com/imrtlDrifter/template/internal/config"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(cfg *config.Configuration, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.Server.Addr,
			Handler:           handler,
			ReadTimeout:       cfg.Server.ReadTimeout,
			ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
			WriteTimeout:      cfg.Server.WriteTimeout,
			IdleTimeout:       cfg.Server.IdleTimeout,
			MaxHeaderBytes:    cfg.Server.MaxHeaderBytes,
		},
	}
}

func (s *Server) Run() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.httpServer.Close()
}
