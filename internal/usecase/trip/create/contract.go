//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/XrystikLK/template/internal/model"
)

type (
	tripRepository interface {
		Save(ctx context.Context, trip model.Trip) error
		AddStatusHistory(ctx context.Context, history model.StatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}

	metrics interface {
		IncTripsCreated(ctx context.Context)
	}
)
