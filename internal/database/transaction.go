package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type transactionKey struct{}

// Executor — операции, которые репозиторий выполняет через пул или транзакцию.
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ExecutorFromContext(ctx context.Context, pool *pgxpool.Pool) Executor {
	if tx, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

type TxManager struct {
	pool    *pgxpool.Pool
	timeout time.Duration
}

func NewTxManager(pool *pgxpool.Pool, timeout time.Duration) *TxManager {
	return &TxManager{pool: pool, timeout: timeout}
}

func (m *TxManager) Do(ctx context.Context, fn func(context.Context) error) (err error) {
	if _, ok := ctx.Value(transactionKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}
	txCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	tx, err := m.pool.BeginTx(txCtx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		// recover перехватывает панику fn, чтобы сначала откатить транзакцию.
		p := recover()
		// Откат вызывается всегда: при ошибке fn, при панике и при выходе через
		// runtime.Goexit. После Commit pgx вернёт ErrTxClosed, его игнорируем.
		// Контекст без отмены нужен, чтобы откат сработал и после отмены запроса.
		rollbackCtx, stop := context.WithTimeout(context.WithoutCancel(txCtx), m.timeout)
		defer stop()
		rollbackErr := tx.Rollback(rollbackCtx)
		if errors.Is(rollbackErr, pgx.ErrTxClosed) {
			rollbackErr = nil
		}
		if p != nil {
			// Паника — баг, её нельзя превращать в обычную ошибку: пробрасываем
			// исходное значение дальше. Ошибку отката вернуть некуда; при сбое
			// отката pgx закрывает соединение, и PostgreSQL сам прерывает транзакцию.
			panic(p)
		}
		if rollbackErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback transaction: %w", rollbackErr))
		}
	}()
	txCtx = context.WithValue(txCtx, transactionKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	if err := tx.Commit(txCtx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
