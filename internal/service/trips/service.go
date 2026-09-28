package trips

import (
	"context"

	"github.com/google/uuid"
	history "github.com/makkssik/template/internal/domain/trip_status_history"
	domaintrips "github.com/makkssik/template/internal/domain/trips"
)

type ITripsService interface {
	CreateTrip(ctx context.Context, opts CreateTripOpts) (*domaintrips.Trip, error)
	GetTrip(ctx context.Context, tripID uuid.UUID) (*domaintrips.Trip, error)
	FinishTrip(ctx context.Context, tripID uuid.UUID) (*domaintrips.Trip, error)
}

type TxManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	tripsRepository         domaintrips.Repository
	statusHistoryRepository history.Repository
	txManager               TxManager
}

type ServiceCfg struct {
	TripsRepository         domaintrips.Repository
	StatusHistoryRepository history.Repository
	TxManager               TxManager
}

func NewService(cfg ServiceCfg) ITripsService {
	return &Service{
		tripsRepository:         cfg.TripsRepository,
		statusHistoryRepository: cfg.StatusHistoryRepository,
		txManager:               cfg.TxManager,
	}
}

type CreateTripOpts struct {
	UserID     uuid.UUID
	DriverID   uuid.UUID
	StartPoint domaintrips.Coordinates
	EndPoint   domaintrips.Coordinates
	Price      int64
}
