package app

import (
	api "github.com/makkssik/template/internal/generated"
	"github.com/makkssik/template/internal/service/checkup"
	"github.com/makkssik/template/internal/service/trips"
)

type Implementation struct {
	api.Unimplemented

	tripsService   trips.ITripsService
	checkupService checkup.ICheckupService
}

type ImplementationConfig struct {
	TripsService   trips.ITripsService
	CheckupService checkup.ICheckupService
}

func New(cfg ImplementationConfig) *Implementation {
	return &Implementation{
		tripsService:   cfg.TripsService,
		checkupService: cfg.CheckupService,
	}
}
