package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/George1487/MalyginGeorge/internal/domain/trip"
	api "github.com/George1487/MalyginGeorge/internal/generated"
	"github.com/George1487/MalyginGeorge/internal/http/domainerror"
	tripCreate "github.com/George1487/MalyginGeorge/internal/http/trip/create"
	tripFinish "github.com/George1487/MalyginGeorge/internal/http/trip/finish"
	tripGet "github.com/George1487/MalyginGeorge/internal/http/trip/get"
	posCreate "github.com/George1487/MalyginGeorge/internal/http/trip/position/create"
	posGet "github.com/George1487/MalyginGeorge/internal/http/trip/position/get"
)

type readinessChecker interface {
	Ping(ctx context.Context) error
}

type Server struct {
	createHandler             *tripCreate.Handler
	getHandler                *tripGet.Handler
	finishHandler             *tripFinish.Handler
	checker                   readinessChecker
	getPositionHandler        *posGet.Handler
	createPositionDataHandler *posCreate.Handler
}

func NewServer(
	createHandler *tripCreate.Handler,
	getHandler *tripGet.Handler,
	finishHandler *tripFinish.Handler,
	checker readinessChecker,
	getPositionHandler *posGet.Handler,
	createPositionDataHandler *posCreate.Handler,
) *Server {
	return &Server{
		createHandler:             createHandler,
		getHandler:                getHandler,
		finishHandler:             finishHandler,
		checker:                   checker,
		getPositionHandler:        getPositionHandler,
		createPositionDataHandler: createPositionDataHandler,
	}
}

func (s *Server) CreateTrip(w http.ResponseWriter, r *http.Request, _ api.CreateTripParams) {
	s.createHandler.Handle(w, r)
}

func (s *Server) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.getHandler.Handle(w, r, tripID)
}

func (s *Server) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.finishHandler.Handle(w, r, tripID)
}

func (s *Server) GetTripPositions(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.getPositionHandler.Handle(w, r, tripID)
}

func (s *Server) SaveTripPosition(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	s.createPositionDataHandler.Handle(w, r, tripID)
}

func (s *Server) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (s *Server) Ready(w http.ResponseWriter, r *http.Request) {
	if s.checker == nil || s.checker.Ping(r.Context()) != nil {
		writeJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func ErrorHandler(w http.ResponseWriter, _ *http.Request, _ error) {
	domainerror.Write(w, trip.ErrInvalidRequest)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

var _ api.ServerInterface = (*Server)(nil)
