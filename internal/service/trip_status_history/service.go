package trip_status_history

import (
	"context"

	domainhistory "github.com/makkssik/template/internal/domain/trip_status_history"
)

type ITripStatusHistoryService interface {
	RegisterStatusChange(ctx context.Context, change domainhistory.StatusChange) error
}

type Service struct {
	statusHistoryRepository domainhistory.Repository
}

type ServiceCfg struct {
	StatusHistoryRepository domainhistory.Repository
}

func NewService(cfg ServiceCfg) ITripStatusHistoryService {
	return &Service{
		statusHistoryRepository: cfg.StatusHistoryRepository,
	}
}
