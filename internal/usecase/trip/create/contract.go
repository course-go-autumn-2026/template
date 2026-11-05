package create

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=create_test

import (
	"context"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type (
	TripRepository interface {
		Save(ctx context.Context, trip appmodel.Trip) error
	}

	TxManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)