package transaction

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toooriii/tori/internal/infra/postgres/querier"
)

type ctxKey struct{}

type Manager struct {
	pool *pgxpool.Pool
	iso  pgx.TxIsoLevel
}

func New(pool *pgxpool.Pool, iso pgx.TxIsoLevel) *Manager {
	return &Manager{pool: pool, iso: iso}
}

func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(ctxKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: m.iso})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		rbErr := tx.Rollback(context.WithoutCancel(ctx))
		if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			slog.Error("rollback failed", "error", rbErr)
		}
	}()

	if err := fn(context.WithValue(ctx, ctxKey{}, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	committed = true
	return nil
}

func (m *Manager) Querier(ctx context.Context) querier.Querier {
	if tx, ok := ctx.Value(ctxKey{}).(pgx.Tx); ok {
		return tx
	}
	return m.pool
}
