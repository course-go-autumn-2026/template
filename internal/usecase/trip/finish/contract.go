//go:generate go tool mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/XrystikLK/template/internal/model"
)

type (
	tripRepository interface {
		GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error)
		Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error)
		AddStatusHistory(ctx context.Context, history model.StatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	logger interface {
		InfoContext(ctx context.Context, msg string, args ...any)
	}

	metrics interface {
		IncTripsCompleted(ctx context.Context)
	}
)
