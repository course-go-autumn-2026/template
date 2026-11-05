package finish

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=finish_test

import (
	"context"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type (
	TripRepository interface {
		GetByID(ctx context.Context, id string) (*appmodel.Trip, error)
		Update(ctx context.Context, trip appmodel.Trip) error
	}
	TxManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)