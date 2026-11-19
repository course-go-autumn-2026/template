package create

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/xotapy-main/courseGo/internal/domain/trip"
	"github.com/xotapy-main/courseGo/internal/model"
)

type Request struct {
	TripID      uuid.UUID
	Coordinates model.Point
	RecordedAt  time.Time
}

type Usecase struct {
	repo      positionRepository
	txManager txManager
	canceller pollCanceller
	now       func() time.Time
}

func New(repo positionRepository, txMgr txManager, canceller pollCanceller) *Usecase {
	return &Usecase{
		repo:      repo,
		txManager: txMgr,
		canceller: canceller,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (u *Usecase) Execute(ctx context.Context, req Request) (*model.Position, error) {
	if err := domaintrip.ValidatePosition(req.Coordinates.Latitude, req.Coordinates.Longitude, req.RecordedAt, u.now()); err != nil {
		return nil, err
	}

	pos := &model.Position{
		TripID:      req.TripID,
		Coordinates: req.Coordinates,
		RecordedAt:  req.RecordedAt.UTC(),
	}

	err := u.txManager.Do(ctx, func(txCtx context.Context) error {
		if err := u.repo.UpdateLastPositionAt(txCtx, pos.TripID, pos.RecordedAt); err != nil {
			return err
		}
		id, err := u.repo.InsertPosition(txCtx, pos)
		if err != nil {
			return err
		}
		pos.ID = id
		return nil
	})
	if err != nil {
		return nil, err
	}

	if u.canceller != nil {
		u.canceller.Cancel(pos.TripID)
	}

	return pos, nil
}
