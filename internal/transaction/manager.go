package transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Tx содержит методы, необходимые менеджеру и репозиториям.
type Tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
	Rollback(context.Context) error
}

// Beginner умеет начинать транзакции.
type Beginner interface {
	Begin(context.Context) (Tx, error)
}

type Manager struct {
	beginner Beginner
}

func NewManager(beginner Beginner) *Manager {
	return &Manager{beginner: beginner}
}

type txContextKey struct{}

// TxFromContext возвращает транзакцию из контекста.
func TxFromContext(ctx context.Context) (Tx, bool) {
	tx, ok := ctx.Value(txContextKey{}).(Tx)
	return tx, ok
}

func (m *Manager) WithinTransaction(
	ctx context.Context,
	fn func(context.Context) error,
) (err error) {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.beginner.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txCtx := context.WithValue(ctx, txContextKey{}, tx)

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(recovered)
		}
	}()

	if err = fn(txCtx); err != nil {
		rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(
				err,
				fmt.Errorf("rollback transaction: %w", rollbackErr),
			)
		}
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
