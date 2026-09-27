package service

import (
	"context"
	"errors"

	"github.com/imrtlDrifter/template/internal/repositories"
)

type TransactionManager interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

type Service struct {
	Trip *TripService
}

func NewService(repos *repositories.Repository, txManager TransactionManager) (*Service, error) {
	if repos == nil {
		return nil, errors.New("repository is nil")
	}
	if txManager == nil {
		return nil, errors.New("transaction manager is nil")
	}

	return &Service{
		Trip: NewTripService(repos, txManager),
	}, nil
}
