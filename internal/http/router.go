// Package router assembles the generated api.ServerInterface from the three trip handlers.
package router

import (
	"context"
	"net/http"
	"time"

	api "github.com/n1ckerr0r/avito-labs/internal/generated"
	"github.com/n1ckerr0r/avito-labs/internal/http/httputil"
	createhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/create"
	finishhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/finish"
	gethttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/get"
	positionhttp "github.com/n1ckerr0r/avito-labs/internal/http/trip/positions"
)

// pinger is satisfied by *pgxpool.Pool without importing internal/infra here.
type pinger interface {
	Ping(ctx context.Context) error
}

type API struct {
	api.Unimplemented

	Positions   *positionhttp.Handler
	create      *createhttp.Handler
	get         *gethttp.Handler
	finish      *finishhttp.Handler
	db          pinger
	pingTimeout time.Duration
}

func New(create *createhttp.Handler, get *gethttp.Handler, finish *finishhttp.Handler, db pinger, pingTimeout time.Duration) *API {
	return &API{create: create, get: get, finish: finish, db: db, pingTimeout: pingTimeout}
}

func (a *API) CreateTrip(w http.ResponseWriter, r *http.Request, params api.CreateTripParams) {
	a.create.Handle(w, r, params)
}

func (a *API) GetTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	a.get.Handle(w, r, tripID)
}

func (a *API) FinishTrip(w http.ResponseWriter, r *http.Request, tripID api.TripId) {
	a.finish.Handle(w, r, tripID)
}

func (a *API) Health(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

func (a *API) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), a.pingTimeout)
	defer cancel()

	if err := a.db.Ping(ctx); err != nil {
		httputil.WriteJSON(w, http.StatusServiceUnavailable, api.HealthResponse{Status: api.Unavailable})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, api.HealthResponse{Status: api.Ok})
}

// ErrorHandler renders request parsing failures as RFC 9457 problems.
func ErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	httputil.WriteProblem(w, r, http.StatusBadRequest, httputil.Problem("invalid_request", "Invalid request", err.Error(), http.StatusBadRequest))
}

func (a *API) CreateTripPosition(w http.ResponseWriter, r *http.Request, id api.TripId) {
	a.Positions.Create(w, r, id)
}

func (a *API) ListTripPositions(w http.ResponseWriter, r *http.Request, id api.TripId) {
	a.Positions.List(w, r, id)
}
