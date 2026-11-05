package http

import (
	"context"
	"encoding/json"
	"net/http"

	api "github.com/eternal-despair/template/internal/generated"
	createTrip "github.com/eternal-despair/template/internal/http/trip/create"
	finishTrip "github.com/eternal-despair/template/internal/http/trip/finish"
	getTrip "github.com/eternal-despair/template/internal/http/trip/get"
)

const statusKey = "status"

type pinger interface {
	Ping(ctx context.Context) error
}

type Server struct {
	createHandler *createTrip.Handler
	getHandler    *getTrip.Handler
	finishHandler *finishTrip.Handler
	pinger        pinger
}

func NewServer(
	createH *createTrip.Handler,
	getH *getTrip.Handler,
	finishH *finishTrip.Handler,
	p pinger,
) *Server {
	return &Server{
		createHandler: createH,
		getHandler:    getH,
		finishHandler: finishH,
		pinger:        p,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	s.createHandler.Handle(w, r, params)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.getHandler.Handle(w, r, tripId)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripId api.TripId) {
	s.finishHandler.Handle(w, r, tripId)
}

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "ok"})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.pinger != nil {
		if err := s.pinger.Ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "database unavailable"})
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{statusKey: "ok"})
}
