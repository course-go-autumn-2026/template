package get

//go:generate mockgen -source=contract.go -destination=mocks_test.go -package=get_test

import (
	"context"

	appmodel "github.com/ggwpfu9/courseAvito/internal/model"
)

type TripRepository interface {
	GetByID(ctx context.Context, id string) (*appmodel.Trip, error)
}