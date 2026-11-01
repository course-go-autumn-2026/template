package checkup

import (
	"context"
	"fmt"
	"time"
)

type IDatabasePinger interface {
	Ping(ctx context.Context) error
}

type ICheckupService interface {
	Check(ctx context.Context) error
}

type Service struct {
	database     IDatabasePinger
	queryTimeout time.Duration
}

type ServiceCfg struct {
	Database     IDatabasePinger
	QueryTimeout time.Duration
}

func NewService(cfg ServiceCfg) ICheckupService {
	return &Service{
		database:     cfg.Database,
		queryTimeout: cfg.QueryTimeout,
	}
}

func (s *Service) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.queryTimeout)
	defer cancel()

	if err := s.database.Ping(ctx); err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	return nil
}
