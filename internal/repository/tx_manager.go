package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}


type QueryExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type txManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &txManager{pool: pool}
}

func (tm *txManager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	// Если транзакция уже открыта выше по стеку вызовов, используем её повторно
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok && tx != nil {
		return fn(ctx)
	}

	// Уровень изоляции: ReadCommitted
	opts := pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	}

	tx, err := tm.pool.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	if err = fn(ctxWithTx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

// getExecutor извлекает активную транзакцию из контекста. Если её нет — возвращает пул
func getExecutor(ctx context.Context, pool *pgxpool.Pool) QueryExecutor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok && tx != nil {
		return tx
	}
	return pool
}
