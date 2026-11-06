package container

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toooriii/tori/internal/config"
	createHandler "github.com/toooriii/tori/internal/http/trip/create"
	finishHandler "github.com/toooriii/tori/internal/http/trip/finish"
	getHandler "github.com/toooriii/tori/internal/http/trip/get"
	"github.com/toooriii/tori/internal/infra/postgres"
	"github.com/toooriii/tori/internal/infra/postgres/transaction"
	idemRepo "github.com/toooriii/tori/internal/repository/idempotency"
	tripRepo "github.com/toooriii/tori/internal/repository/trip"
	createUC "github.com/toooriii/tori/internal/usecase/trip/create"
	finishUC "github.com/toooriii/tori/internal/usecase/trip/finish"
	getUC "github.com/toooriii/tori/internal/usecase/trip/get"
)

type Container struct {
	Log  *slog.Logger
	Pool *pgxpool.Pool
	Tx   *transaction.Manager

	CreateTrip *createHandler.Handler
	GetTrip    *getHandler.Handler
	FinishTrip *finishHandler.Handler

	Ping func(ctx context.Context) error
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*Container, error) {
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	tx := transaction.New(pool, pgx.ReadCommitted)

	trips := tripRepo.New(tx)
	idem := idemRepo.New(tx)

	create := createUC.New(trips, idem, tx)
	get := getUC.New(trips)
	finish := finishUC.New(trips, tx)

	return &Container{
		Log:        log,
		Pool:       pool,
		Tx:         tx,
		CreateTrip: createHandler.New(create),
		GetTrip:    getHandler.New(get),
		FinishTrip: finishHandler.New(finish),
		Ping:       pool.Ping,
	}, nil
}

func (c *Container) Close() {
	if c.Pool != nil {
		c.Pool.Close()
	}
}
