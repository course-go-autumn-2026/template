package trip_status_history

import (
	"context"
	"fmt"

	domainhistory "github.com/makkssik/template/internal/domain/trip_status_history"
)

func (s *Service) RegisterStatusChange(ctx context.Context, change domainhistory.StatusChange) error {
	if err := s.statusHistoryRepository.RegisterStatusChange(ctx, change); err != nil {
		return fmt.Errorf("register status change: %w", err)
	}
	return nil
}
