package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type transactionContextKey struct{}

type TransactionManager struct {
	pool             *pgxpool.Pool
	operationTimeout time.Duration
}

func NewTransactionManager(pool *pgxpool.Pool, operationTimeout time.Duration) *TransactionManager {
	return &TransactionManager{
		pool:             pool,
		operationTimeout: operationTimeout,
	}
}

func (m *TransactionManager) Do(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := transactionFromContext(ctx); ok {
		return fn(ctx)
	}

	beginCtx, cancelBegin := context.WithTimeout(ctx, m.operationTimeout)
	tx, err := m.pool.BeginTx(beginCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	cancelBegin()

	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}

		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), m.operationTimeout)
		defer cancel()

		_ = tx.Rollback(rollbackCtx)
	}()

	txCtx := context.WithValue(ctx, transactionContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	commitCtx, cancelCommit := context.WithTimeout(ctx, m.operationTimeout)
	err = tx.Commit(commitCtx)
	cancelCommit()

	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (m *TransactionManager) Executor(ctx context.Context) DBTX {
	return GetExecutor(ctx, m.pool)
}

func GetExecutor(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := transactionFromContext(ctx); ok {
		return tx
	}

	return pool
}

func transactionFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx)
	return tx, ok
}
