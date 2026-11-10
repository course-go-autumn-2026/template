package finish

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"

	domaintrip "github.com/murmyauuu/template/internal/domain/trip"
)

type Usecase struct {
	repo      tripRepository
	tx        txManager
	history   historyRepository
	logger    *slog.Logger
	completed metric.Int64Counter
}

func New(repo tripRepository, tx txManager, history historyRepository, logger *slog.Logger) *Usecase {
	meter := otel.Meter("trip-service")
	completed, _ := meter.Int64Counter("tripgo.trips.completed")
	return &Usecase{repo: repo, tx: tx, history: history, logger: logger, completed: completed}
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
		u.completed.Add(ctx, 1)
		if u.logger != nil {
			u.logger.InfoContext(ctx, "trip completed", "trip_id", id.String())
		}
		from := domaintrip.StatusActive
		return u.history.Append(ctx, id, &from, domaintrip.StatusCompleted, finishedAt)
	})
	if err != nil {
		return domaintrip.Trip{}, err
	}
	return finished, nil
}
