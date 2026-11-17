package transaction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager struct {
	logger  *slog.Logger
	pool    *pgxpool.Pool
	timeout time.Duration
}

type txKey struct{}

func NewTxManager(pool *pgxpool.Pool, timeout time.Duration, logger *slog.Logger) *TxManager {
	return &TxManager{
		pool:    pool,
		logger:  logger,
		timeout: timeout,
	}
}

func (m *TxManager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	_, ok := ctx.Value(txKey{}).(pgx.Tx)
	if ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		rollbackCtx, cancel := context.WithTimeout(
			context.WithoutCancel(ctx),
			m.timeout,
		)
		defer cancel()

		if err := tx.Rollback(rollbackCtx); err != nil &&
			!errors.Is(err, pgx.ErrTxClosed) {
			m.logger.ErrorContext(
				rollbackCtx,
				"rollback transaction failed",
				"error", err,
			)
		}
	}()

	txCtx := context.WithValue(ctx, txKey{}, tx)

	err = fn(txCtx)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
