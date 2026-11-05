package trip

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xotapy-main/courseGo/internal/model"
)

type Repository interface {
	Save(ctx context.Context, t *model.Trip) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Trip, error)
	Finish(ctx context.Context, id uuid.UUID, finishedAt time.Time) (*model.Trip, error)
	InsertHistory(ctx context.Context, record *model.TripStatusHistory) error
}
