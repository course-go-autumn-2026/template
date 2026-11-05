package container

import (
	"github.com/jackc/pgx/v5/pgxpool"

	tx "github.com/ggwpfu9/courseAvito/internal/infra/postgres/transaction"
	tripRepoPkg "github.com/ggwpfu9/courseAvito/internal/repository/trip"
	
	createUc "github.com/ggwpfu9/courseAvito/internal/usecase/trip/create"
	getUc "github.com/ggwpfu9/courseAvito/internal/usecase/trip/get"
	finishUc "github.com/ggwpfu9/courseAvito/internal/usecase/trip/finish"
)

type Container struct {
	CreateTrip *createUc.UseCase
	GetTrip    *getUc.UseCase
	FinishTrip *finishUc.UseCase
}

func New(pool *pgxpool.Pool) *Container {
	txManager := tx.NewTxManager(pool)

	tRepo := tripRepoPkg.NewRepo(pool)

	return &Container{
		CreateTrip: createUc.New(tRepo, txManager),
		GetTrip:    getUc.New(tRepo),
		FinishTrip: finishUc.New(tRepo, txManager),
	}
}