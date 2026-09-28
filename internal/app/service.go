package app

import (
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	api "github.com/makkssik/template/internal/generated"
	"github.com/makkssik/template/internal/service/trips"
)

type Implementation struct {
	api.Unimplemented

	pool         *pgxpool.Pool
	queryTimeout time.Duration
	tripsService trips.ITripsService
}

type ImplementationConfig struct {
	Pool         *pgxpool.Pool
	QueryTimeout time.Duration
	TripsService trips.ITripsService
}

func New(cfg ImplementationConfig) *Implementation {
	return &Implementation{
		pool:         cfg.Pool,
		queryTimeout: cfg.QueryTimeout,
		tripsService: cfg.TripsService,
	}
}
