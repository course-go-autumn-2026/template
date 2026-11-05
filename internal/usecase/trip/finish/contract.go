//go:generate mockgen -source ${GOFILE} -destination mocks_test.go -package${GOPACKAGE}_test
package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type (
	tripRepository interface {
		GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error)
		Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error)
		InsertHistory(ctx context.Context, record *model.TripStatusHistory) error
	}

	txManager interface {
		Do(ctx context.Context, fn func(ctx context.Context) error) error
	}
)
