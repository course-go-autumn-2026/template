//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/eternal-despair/template/internal/model"
)

type (
	tripRepository interface {
		Create(ctx context.Context, trip *model.Trip) error
		CreateStatusHistory(ctx context.Context, history *model.TripStatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		TripCreated(ctx context.Context)
	}
)
