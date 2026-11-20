package transaction

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxTxManager struct {
	pool *pgxpool.Pool
}

func NewPgxTxManager(pool *pgxpool.Pool) *PgxTxManager {
	return &PgxTxManager{pool: pool}
}

type TxKey struct{}

func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(TxKey{}).(pgx.Tx)
	return tx, ok
}

func (txm *PgxTxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	_, ok := TxFromContext(ctx)
	if ok {
		return fn(ctx)
	}
	tr, err := txm.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("pgx.TxManager.Begin: %w", err)
	}

	defer func() {
		p := recover()
		if p != nil {
			rollbackCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = tr.Rollback(rollbackCtx)
			panic(p)
		}
	}()

	err = fn(context.WithValue(ctx, TxKey{}, tr))
	if err != nil {
		rbErr := tr.Rollback(ctx)
		if rbErr != nil {
			return errors.Join(
				fmt.Errorf("pgx.TxManager: functions error : %w", err),
				fmt.Errorf("pgx.TxManager.Rollback failed: %w", rbErr),
			)
		}
		return fmt.Errorf("pgx.TxManager.Do: %w", err)
	}
	err = tr.Commit(ctx)
	if err != nil {
		return fmt.Errorf("pgx.TxManager.Commit: %w", err)
	}
	return nil
}
