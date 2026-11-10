package finish

import (
	"context"
	"time"

	"github.com/google/uuid"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type Usecase struct {
	repo    tripRepository
	tx      txManager
	history historyRepository
}

func New(repo tripRepository, tx txManager, history historyRepository) *Usecase {
	return &Usecase{repo: repo, tx: tx, history: history}
}

func (u *Usecase) Execute(ctx context.Context, id uuid.UUID) (domaintrip.Trip, error) {
	finishedAt := time.Now().UTC().Truncate(time.Microsecond)
	var finished domaintrip.Trip
	err := u.tx.Do(ctx, func(ctx context.Context) error {
		var err error
		finished, err = u.repo.Finish(ctx, id, finishedAt)
		if err != nil {
			return err
		}
		from := domaintrip.StatusActive
		return u.history.Append(ctx, id, &from, domaintrip.StatusCompleted, finishedAt)
	})
	if err != nil {
		return domaintrip.Trip{}, err
	}
	return finished, nil
}
