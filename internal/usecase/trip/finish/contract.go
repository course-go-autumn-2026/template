//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package ${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/eternal-despair/template/internal/model"
)

type (
	tripRepository interface {
		Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error)
		CreateStatusHistory(ctx context.Context, history *model.TripStatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}

	metrics interface {
		TripCompleted(ctx context.Context)
	}
)
